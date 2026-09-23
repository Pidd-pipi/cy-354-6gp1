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

// reportRepository is the data access contract for report rows.
type reportRepository interface {
	Create(ctx context.Context, rp *model.Report) error
	FindByID(ctx context.Context, id uint) (*model.Report, error)
	FindByProductAndReporter(ctx context.Context, productID, reporterID uint) (*model.Report, error)
	ListByReporter(ctx context.Context, reporterID uint) ([]model.Report, error)
	ListByStatus(ctx context.Context, status string, page, pageSize int) ([]model.Report, int64, error)
	Resolve(ctx context.Context, id, handlerID uint, status, remark string, handledAt interface{}) error
	Transaction(ctx context.Context, fn func(txCtx context.Context) error) error
}

// reportProductRepository is the product access contract needed by reports.
type reportProductRepository interface {
	FindByID(ctx context.Context, id uint) (*model.Product, error)
	UpdateStatusIfOnSale(ctx context.Context, id uint, status string) error
}

// ReportService manages student product reports and admin handling.
type ReportService struct {
	reports  reportRepository
	products reportProductRepository
	logger   *slog.Logger
}

// NewReportService wires the report service dependencies.
func NewReportService(reports reportRepository, products reportProductRepository, logger *slog.Logger) *ReportService {
	return &ReportService{reports: reports, products: products, logger: logger}
}

// Create records a report against an on-sale product. Repeated submission by
// the same student for the same product returns the original record.
func (s *ReportService) Create(ctx context.Context, reporterID uint, req *dto.CreateReportRequest) (*model.Report, error) {
	if !constants.IsReportReason(req.Reason) {
		return nil, util.NewAppError(400, constants.CodeValidation, "举报原因不合法", nil)
	}
	product, err := s.products.FindByID(ctx, req.ProductID)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("report[product=%d] product lookup: %w", req.ProductID, err), 404, constants.CodeNotFound, constants.MsgNotFound)
	}
	if product.SellerID == reporterID {
		return nil, util.NewAppError(400, constants.CodeBadRequest, constants.MsgReportTarget, nil)
	}
	// Duplicate submission returns the original record unchanged.
	if existing, err := s.reports.FindByProductAndReporter(ctx, req.ProductID, reporterID); err == nil {
		s.logger.Info(fmt.Sprintf(constants.LogReportDuplicateSubmission, existing.ID, reporterID, req.ProductID))
		return existing, nil
	} else if !errors.Is(err, util.ErrNotFound) {
		return nil, util.WrapAppError(fmt.Errorf("report[product=%d] duplicate lookup: %w", req.ProductID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	if product.Status != constants.ProductStatusOnSale {
		return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgProductNotOnSale, nil)
	}
	rp := &model.Report{
		ProductID: req.ProductID, ReporterID: reporterID,
		Reason: req.Reason, Description: req.Description,
		Status: constants.ReportStatusPending,
	}
	if err := s.reports.Create(ctx, rp); err != nil {
		// Concurrent first submissions collide on the unique index: return the
		// original record so the response stays idempotent.
		if existing, findErr := s.reports.FindByProductAndReporter(ctx, req.ProductID, reporterID); findErr == nil {
			s.logger.Info(fmt.Sprintf(constants.LogReportDuplicateSubmission, existing.ID, reporterID, req.ProductID))
			return existing, nil
		}
		s.logger.Error(fmt.Sprintf(constants.LogReportCreateFailed, req.ProductID, reporterID, err))
		return nil, util.WrapAppError(fmt.Errorf("report[product=%d reporter=%d] create: %w", req.ProductID, reporterID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	s.logger.Info(fmt.Sprintf(constants.LogReportCreateSuccess, rp.ID, req.ProductID, reporterID, req.Reason))
	return rp, nil
}

// ListMine returns reports submitted by the current student.
func (s *ReportService) ListMine(ctx context.Context, reporterID uint) ([]model.Report, error) {
	items, err := s.reports.ListByReporter(ctx, reporterID)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("report[reporter=%d] list mine: %w", reporterID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	return items, nil
}

// ListPending returns reports for the admin console, optionally filtered by status.
func (s *ReportService) ListPending(ctx context.Context, q *dto.ListReportQuery) (*dto.PageResult, error) {
	q.Normalize()
	status := q.Status
	if status == "" {
		status = constants.ReportStatusPending
	}
	if !constants.IsReportStatus(status) {
		return nil, util.NewAppError(400, constants.CodeValidation, "举报状态不合法", nil)
	}
	items, total, err := s.reports.ListByStatus(ctx, status, q.Page, q.PageSize)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("report admin list[status=%s]: %w", status, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	return &dto.PageResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// Handle resolves a report as admin. A take_down changes product status and the
// report result together in one transaction; if the product was sold, removed,
// or another admin handled the report first, both sides stay untouched.
func (s *ReportService) Handle(ctx context.Context, adminID uint, reportID uint, req *dto.HandleReportRequest) (*model.Report, error) {
	if !constants.IsReportAction(req.Action) {
		return nil, util.NewAppError(400, constants.CodeValidation, "举报处理方式不合法", nil)
	}
	rp, err := s.reports.FindByID(ctx, reportID)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("report[id=%d] handle find: %w", reportID, err), 404, constants.CodeNotFound, constants.MsgNotFound)
	}
	if rp.Status != constants.ReportStatusPending {
		return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgReportHandled, nil)
	}

	var finalStatus string
	switch req.Action {
	case constants.ReportActionTakeDown:
		// 下架要求商品仍在售；已售出/已下架则拒绝处理，两边都不变。
		product, err := s.products.FindByID(ctx, rp.ProductID)
		if err != nil {
			return nil, util.WrapAppError(fmt.Errorf("report[id=%d] product lookup: %w", reportID, err), 404, constants.CodeNotFound, constants.MsgNotFound)
		}
		if product.Status != constants.ProductStatusOnSale {
			s.logger.Warn(fmt.Sprintf(constants.LogReportHandleRejected, reportID, adminID, "product_status="+product.Status))
			return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgProductNotOnSale, nil)
		}
		finalStatus = constants.ReportStatusTakenDown
	case constants.ReportActionReject:
		// 驳回保留商品，仅记录驳回原因；商品是否在售均不影响驳回。
		finalStatus = constants.ReportStatusRejected
	}

	now := time.Now()
	txErr := s.reports.Transaction(ctx, func(txCtx context.Context) error {
		if req.Action == constants.ReportActionTakeDown {
			if err := s.products.UpdateStatusIfOnSale(txCtx, rp.ProductID, constants.ProductStatusRemoved); err != nil {
				return err
			}
		}
		return s.reports.Resolve(txCtx, reportID, adminID, finalStatus, req.Remark, now)
	})
	if txErr != nil {
		switch {
		case errors.Is(txErr, util.ErrConflict), errors.Is(txErr, repository.ErrAlreadyHandled):
			// Another admin won the race, or the product just changed state:
			// reject the request and leave both sides unchanged (tx rolled back).
			s.logger.Warn(fmt.Sprintf(constants.LogReportHandleRejected, reportID, adminID, txErr.Error()))
			return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgReportHandled, nil)
		default:
			s.logger.Error(fmt.Sprintf(constants.LogReportHandleFailed, reportID, adminID, txErr))
			return nil, util.WrapAppError(fmt.Errorf("report[id=%d] admin[%d] handle: %w", reportID, adminID, txErr), 500, constants.CodeInternalError, constants.MsgInternalError)
		}
	}
	s.logger.Info(fmt.Sprintf(constants.LogReportHandleSuccess, reportID, req.Action))
	rp.Status = finalStatus
	rp.HandlerID = &adminID
	rp.HandleRemark = req.Remark
	rp.HandledAt = &now
	return rp, nil
}
