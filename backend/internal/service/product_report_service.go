package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/repository"
	"github.com/lp/campus-market/internal/util"
)

// productReportStore is the data access contract for product report rows.
type productReportStore interface {
	Create(ctx context.Context, rp *model.ProductReport) error
	FindByID(ctx context.Context, id uint) (*model.ProductReport, error)
	FindByIDForUpdate(ctx context.Context, id uint) (*model.ProductReport, error)
	FindPendingByProductAndReporter(ctx context.Context, productID, reporterID uint) (*model.ProductReport, error)
	ListByReporter(ctx context.Context, reporterID uint) ([]model.ProductReport, error)
	ListByStatus(ctx context.Context, status string) ([]model.ProductReport, error)
	Complete(ctx context.Context, id, handlerID uint, status, remark string, ts interface{}) error
	Transaction(ctx context.Context, fn func(txCtx context.Context) error) error
}

// reportProductStore is the product access contract used while reporting.
type reportProductStore interface {
	FindByID(ctx context.Context, id uint) (*model.Product, error)
	FindByIDForUpdate(ctx context.Context, id uint) (*model.Product, error)
	FindByIDs(ctx context.Context, ids []uint) ([]model.Product, error)
	UpdateStatusIfOnSale(ctx context.Context, id uint, status string) error
}

// reportUserStore is the user lookup contract used for reporter display names.
type reportUserStore interface {
	FindByIDs(ctx context.Context, ids []uint) ([]model.User, error)
}

// ProductReportService manages student product reports and admin handling.
type ProductReportService struct {
	reports  productReportStore
	products reportProductStore
	users    reportUserStore
	logger   *slog.Logger
}

// NewProductReportService wires the product report service dependencies.
func NewProductReportService(reports productReportStore, products reportProductStore, users reportUserStore, logger *slog.Logger) *ProductReportService {
	return &ProductReportService{reports: reports, products: products, users: users, logger: logger}
}

// Submit creates a pending report for an on-sale product. When the reporter
// already owns a pending record for the product, that original record is
// returned instead of creating a new one.
func (s *ProductReportService) Submit(ctx context.Context, reporter *model.User, req *dto.CreateProductReportRequest) (*dto.ProductReportView, error) {
	if !constants.IsReportReason(req.Reason) {
		return nil, util.NewAppError(400, constants.CodeValidation, constants.MsgReportReasonInvalid, nil)
	}
	product, err := s.products.FindByID(ctx, req.ProductID)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("product_report[product=%d] target lookup: %w", req.ProductID, err), 404, constants.CodeNotFound, constants.MsgReportTarget)
	}
	if product.SellerID == reporter.ID {
		return nil, util.NewAppError(400, constants.CodeBadRequest, constants.MsgReportSelf, nil)
	}
	// An existing pending record always wins and is returned as-is, even if the
	// product has since been sold or taken down by another flow.
	if existing, err := s.reports.FindPendingByProductAndReporter(ctx, req.ProductID, reporter.ID); err == nil {
		s.logger.Info(fmt.Sprintf(constants.LogProductReportSubmitDuplicate, existing.ID, req.ProductID, reporter.ID))
		return s.toView(ctx, existing, true)
	} else if !errors.Is(err, util.ErrNotFound) {
		return nil, util.WrapAppError(fmt.Errorf("product_report[product=%d] dedup lookup: %w", req.ProductID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	if product.Status != constants.ProductStatusOnSale {
		return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgProductNotOnSale, nil)
	}
	dedupKey := reportDedupKey(req.ProductID, reporter.ID)
	rp := &model.ProductReport{
		ProductID: req.ProductID, ReporterID: reporter.ID,
		Reason: req.Reason, Description: req.Description,
		Status: constants.ReportStatusPending, DedupKey: &dedupKey,
	}
	if err := s.reports.Create(ctx, rp); err != nil {
		if repository.IsDuplicateKeyError(err) {
			existing, lookupErr := s.reports.FindPendingByProductAndReporter(ctx, req.ProductID, reporter.ID)
			if lookupErr != nil {
				return nil, util.WrapAppError(fmt.Errorf("product_report[product=%d] duplicated lookup: %w", req.ProductID, lookupErr), 500, constants.CodeInternalError, constants.MsgInternalError)
			}
			s.logger.Info(fmt.Sprintf(constants.LogProductReportSubmitDuplicate, existing.ID, req.ProductID, reporter.ID))
			return s.toView(ctx, existing, true)
		}
		s.logger.Error(fmt.Sprintf(constants.LogProductReportSubmitFailed, req.ProductID, reporter.ID, err))
		return nil, util.WrapAppError(fmt.Errorf("product_report[product=%d reporter=%d] create: %w", req.ProductID, reporter.ID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	s.logger.Info(fmt.Sprintf(constants.LogProductReportSubmitSuccess, rp.ID, req.ProductID, reporter.ID))
	return s.toView(ctx, rp, false)
}

// ListMine returns the reports submitted by the current student.
func (s *ProductReportService) ListMine(ctx context.Context, reporterID uint) ([]*dto.ProductReportView, error) {
	items, err := s.reports.ListByReporter(ctx, reporterID)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("product_report[reporter=%d] list: %w", reporterID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	return s.toViews(ctx, items)
}

// ListPending returns the pending report queue for admins.
func (s *ProductReportService) ListPending(ctx context.Context) ([]*dto.ProductReportView, error) {
	items, err := s.reports.ListByStatus(ctx, constants.ReportStatusPending)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("product_report admin list pending: %w", err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	return s.toViews(ctx, items)
}

// Handle applies an admin decision. A take-down atomically flips the product
// to removed together with the report result; when the product is already
// sold/removed or another admin handled the report first, it is rejected with
// both sides untouched. A rejection keeps the product on sale and records why.
func (s *ProductReportService) Handle(ctx context.Context, admin *model.User, reportID uint, req *dto.HandleProductReportRequest) (*dto.ProductReportView, error) {
	if !constants.IsReportAction(req.Action) {
		return nil, util.NewAppError(400, constants.CodeValidation, constants.MsgReportActionInvalid, nil)
	}
	if req.Action == constants.ReportActionReject && req.Remark == "" {
		return nil, util.NewAppError(400, constants.CodeValidation, constants.MsgReportRemarkRequired, nil)
	}
	rp, err := s.reports.FindByID(ctx, reportID)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("product_report[id=%d] handle find: %w", reportID, err), 404, constants.CodeNotFound, constants.MsgNotFound)
	}
	now := time.Now()
	var finalStatus string
	switch req.Action {
	case constants.ReportActionTakeDown:
		finalStatus = constants.ReportStatusTakenDown
		err = s.reports.Transaction(ctx, func(txCtx context.Context) error {
			lockedReport, lockErr := s.reports.FindByIDForUpdate(txCtx, reportID)
			if lockErr != nil {
				return lockErr
			}
			if lockedReport.Status != constants.ReportStatusPending {
				return util.ErrConflict
			}
			lockedProduct, productErr := s.products.FindByIDForUpdate(txCtx, rp.ProductID)
			if productErr != nil {
				return productErr
			}
			if lockedProduct.Status != constants.ProductStatusOnSale {
				return util.ErrConflict
			}
			if updateErr := s.products.UpdateStatusIfOnSale(txCtx, rp.ProductID, constants.ProductStatusRemoved); updateErr != nil {
				return updateErr
			}
			return s.reports.Complete(txCtx, reportID, admin.ID, finalStatus, req.Remark, now)
		})
	case constants.ReportActionReject:
		finalStatus = constants.ReportStatusRejected
		err = s.reports.Complete(ctx, reportID, admin.ID, finalStatus, req.Remark, now)
	}
	if err != nil {
		if errors.Is(err, util.ErrConflict) {
			s.logger.Warn(fmt.Sprintf(constants.LogProductReportHandleConflict, reportID, admin.ID, req.Action))
			if req.Action == constants.ReportActionTakeDown {
				return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgReportProductGone, nil)
			}
			return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgReportHandled, nil)
		}
		if errors.Is(err, util.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, constants.MsgNotFound, nil)
		}
		return nil, util.WrapAppError(fmt.Errorf("product_report[id=%d] %s handle: %w", reportID, req.Action, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	if req.Action == constants.ReportActionTakeDown {
		s.logger.Info(fmt.Sprintf(constants.LogProductReportTakeDownSuccess, reportID, rp.ProductID, admin.ID))
	} else {
		s.logger.Info(fmt.Sprintf(constants.LogProductReportRejectSuccess, reportID, admin.ID))
	}
	s.logger.Info(fmt.Sprintf(constants.LogReportHandleSuccess, reportID, req.Action))
	rp.Status = finalStatus
	rp.HandlerID = &admin.ID
	rp.HandleRemark = req.Remark
	rp.HandledAt = &now
	return s.toView(ctx, rp, false)
}

// toView enriches one report with product and reporter display fields.
func (s *ProductReportService) toView(ctx context.Context, rp *model.ProductReport, duplicated bool) (*dto.ProductReportView, error) {
	views, err := s.toViews(ctx, []model.ProductReport{*rp})
	if err != nil {
		return nil, err
	}
	if len(views) == 0 {
		return nil, util.NewAppError(500, constants.CodeInternalError, constants.MsgInternalError, nil)
	}
	views[0].Duplicated = duplicated
	return views[0], nil
}

// toViews enriches report rows with product and reporter display fields.
func (s *ProductReportService) toViews(ctx context.Context, items []model.ProductReport) ([]*dto.ProductReportView, error) {
	productIDs := make([]uint, 0, len(items))
	reporterIDs := make([]uint, 0, len(items))
	seen := map[uint]struct{}{}
	for _, rp := range items {
		if _, ok := seen[rp.ProductID]; !ok {
			productIDs = append(productIDs, rp.ProductID)
			seen[rp.ProductID] = struct{}{}
		}
	}
	seen = map[uint]struct{}{}
	for _, rp := range items {
		if _, ok := seen[rp.ReporterID]; !ok {
			reporterIDs = append(reporterIDs, rp.ReporterID)
			seen[rp.ReporterID] = struct{}{}
		}
	}
	products, err := s.products.FindByIDs(ctx, productIDs)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("product_report enrich products: %w", err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	users, err := s.users.FindByIDs(ctx, reporterIDs)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("product_report enrich reporters: %w", err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	productMap := make(map[uint]model.Product, len(products))
	for _, p := range products {
		productMap[p.ID] = p
	}
	userMap := make(map[uint]model.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}
	out := make([]*dto.ProductReportView, 0, len(items))
	for _, rp := range items {
		v := &dto.ProductReportView{
			ID: rp.ID, ProductID: rp.ProductID, ReporterID: rp.ReporterID,
			Reason: rp.Reason, Description: rp.Description, Status: rp.Status,
			HandlerID: rp.HandlerID, HandleRemark: rp.HandleRemark,
			HandledAt: rp.HandledAt, CreatedAt: rp.CreatedAt,
		}
		if p, ok := productMap[rp.ProductID]; ok {
			v.ProductTitle = p.Title
			v.ProductStatus = p.Status
		}
		if u, ok := userMap[rp.ReporterID]; ok {
			v.ReporterName = u.Nickname
		}
		out = append(out, v)
	}
	return out, nil
}

// reportDedupKey renders the pending-report unique key for a product/reporter pair.
func reportDedupKey(productID, reporterID uint) string {
	return fmt.Sprintf("%d:%d", productID, reporterID)
}
