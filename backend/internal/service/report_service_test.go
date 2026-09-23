package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/repository"
	"github.com/lp/campus-market/internal/util"
)

type fakeReportRepo struct {
	reports map[uint]*model.Report
	nextID  uint
	// resolveFail, when non-nil, is returned by Resolve (simulates race loss).
	resolveFail error
	// products, when set, participates in fake transaction rollback snapshots.
	products *fakeReportProductRepo
}

func newFakeReportRepo() *fakeReportRepo {
	return &fakeReportRepo{reports: map[uint]*model.Report{}, nextID: 1}
}

func (f *fakeReportRepo) Create(_ context.Context, rp *model.Report) error {
	for _, existing := range f.reports {
		if existing.ProductID == rp.ProductID && existing.ReporterID == rp.ReporterID {
			return util.ErrConflict
		}
	}
	rp.ID = f.nextID
	f.nextID++
	cp := *rp
	f.reports[rp.ID] = &cp
	return nil
}

func (f *fakeReportRepo) FindByID(_ context.Context, id uint) (*model.Report, error) {
	if rp, ok := f.reports[id]; ok {
		cp := *rp
		return &cp, nil
	}
	return nil, util.ErrNotFound
}

func (f *fakeReportRepo) FindByProductAndReporter(_ context.Context, productID, reporterID uint) (*model.Report, error) {
	for _, rp := range f.reports {
		if rp.ProductID == productID && rp.ReporterID == reporterID {
			cp := *rp
			return &cp, nil
		}
	}
	return nil, util.ErrNotFound
}

func (f *fakeReportRepo) ListByReporter(_ context.Context, reporterID uint) ([]model.Report, error) {
	var out []model.Report
	for _, rp := range f.reports {
		if rp.ReporterID == reporterID {
			out = append(out, *rp)
		}
	}
	return out, nil
}

func (f *fakeReportRepo) ListByStatus(_ context.Context, status string, _, _ int) ([]model.Report, int64, error) {
	var out []model.Report
	for _, rp := range f.reports {
		if status == "" || rp.Status == status {
			out = append(out, *rp)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeReportRepo) Resolve(_ context.Context, id, handlerID uint, status, remark string, handledAt interface{}) error {
	if f.resolveFail != nil {
		return f.resolveFail
	}
	rp, ok := f.reports[id]
	if !ok {
		return util.ErrNotFound
	}
	if rp.Status != constants.ReportStatusPending {
		return repository.ErrAlreadyHandled
	}
	rp.Status = status
	rp.HandlerID = &handlerID
	rp.HandleRemark = remark
	if t, ok := handledAt.(time.Time); ok {
		rp.HandledAt = &t
	}
	return nil
}

func (f *fakeReportRepo) Transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	// Snapshot state so a failed fake transaction rolls back like a real one.
	reportSnap := map[uint]model.Report{}
	for id, rp := range f.reports {
		reportSnap[id] = *rp
	}
	var productSnap map[uint]string
	if f.products != nil {
		productSnap = map[uint]string{}
		for id, p := range f.products.products {
			productSnap[id] = p.Status
		}
	}
	if err := fn(ctx); err != nil {
		f.reports = map[uint]*model.Report{}
		for id, rp := range reportSnap {
			cp := rp
			f.reports[id] = &cp
		}
		if f.products != nil {
			for id, status := range productSnap {
				f.products.products[id].Status = status
			}
		}
		return err
	}
	return nil
}

type fakeReportProductRepo struct {
	products map[uint]*model.Product
	// updateFail, when non-nil, is returned by UpdateStatusIfOnSale.
	updateFail error
}

func newFakeReportProductRepo(products ...*model.Product) *fakeReportProductRepo {
	m := map[uint]*model.Product{}
	for _, p := range products {
		m[p.ID] = p
	}
	return &fakeReportProductRepo{products: m}
}

func (f *fakeReportProductRepo) FindByID(_ context.Context, id uint) (*model.Product, error) {
	if p, ok := f.products[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, util.ErrNotFound
}

func (f *fakeReportProductRepo) UpdateStatusIfOnSale(_ context.Context, id uint, status string) error {
	if f.updateFail != nil {
		return f.updateFail
	}
	p, ok := f.products[id]
	if !ok {
		return util.ErrNotFound
	}
	if p.Status != constants.ProductStatusOnSale {
		return util.ErrConflict
	}
	p.Status = status
	return nil
}

func newReportService() (*ReportService, *fakeReportRepo, *fakeReportProductRepo) {
	products := newFakeReportProductRepo(
		&model.Product{ID: 1, SellerID: 2, Title: "在售商品", Status: constants.ProductStatusOnSale},
		&model.Product{ID: 2, SellerID: 2, Title: "已售商品", Status: constants.ProductStatusSold},
		&model.Product{ID: 3, SellerID: 2, Title: "卖家已下架", Status: constants.ProductStatusRemoved},
	)
	reports := newFakeReportRepo()
	reports.products = products
	return NewReportService(reports, products, slog.Default()), reports, products
}

func TestReportServiceCreate(t *testing.T) {
	svc, _, _ := newReportService()
	tests := []struct {
		name       string
		reporterID uint
		productID  uint
		reason     string
		wantErr    bool
		wantSameID bool
	}{
		{name: "valid fake description", reporterID: 10, productID: 1, reason: constants.ReportReasonFakeDescription, wantErr: false},
		{name: "valid prohibited item", reporterID: 11, productID: 1, reason: constants.ReportReasonProhibitedItem, wantErr: false},
		{name: "invalid reason", reporterID: 12, productID: 1, reason: "spam", wantErr: true},
		{name: "product missing", reporterID: 13, productID: 99, reason: constants.ReportReasonFakeDescription, wantErr: true},
		{name: "cannot report own product", reporterID: 2, productID: 1, reason: constants.ReportReasonProhibitedItem, wantErr: true},
		{name: "product already sold", reporterID: 14, productID: 2, reason: constants.ReportReasonFakeDescription, wantErr: true},
	}
	var firstID uint
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rp, err := svc.Create(context.Background(), tt.reporterID, &dto.CreateReportRequest{
				ProductID: tt.productID, Reason: tt.reason, Description: "情况说明",
			})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if i == 0 {
				firstID = rp.ID
			}
			if tt.wantSameID && rp.ID != firstID {
				t.Fatalf("expected original report id %d, got %d", firstID, rp.ID)
			}
		})
	}

	// 同一人对同一商品重复提交：返回原记录，不新增。
	again, err := svc.Create(context.Background(), 10, &dto.CreateReportRequest{
		ProductID: 1, Reason: constants.ReportReasonProhibitedItem, Description: "换个理由",
	})
	if err != nil {
		t.Fatalf("duplicate submission should return original, got error: %v", err)
	}
	if again.ID != firstID {
		t.Fatalf("expected original report id %d, got %d", firstID, again.ID)
	}
	if again.Reason != constants.ReportReasonFakeDescription {
		t.Fatalf("original record must stay unchanged, reason=%s", again.Reason)
	}
}

func TestReportServiceHandle(t *testing.T) {
	ctx := context.Background()

	t.Run("take down succeeds together", func(t *testing.T) {
		svc, reports, products := newReportService()
		rp, err := svc.Create(ctx, 10, &dto.CreateReportRequest{ProductID: 1, Reason: constants.ReportReasonProhibitedItem})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		got, err := svc.Handle(ctx, 99, rp.ID, &dto.HandleReportRequest{Action: constants.ReportActionTakeDown, Remark: "违禁物品下架"})
		if err != nil {
			t.Fatalf("handle: %v", err)
		}
		if got.Status != constants.ReportStatusTakenDown {
			t.Fatalf("report status = %s, want taken_down", got.Status)
		}
		if products.products[1].Status != constants.ProductStatusRemoved {
			t.Fatalf("product status = %s, want removed", products.products[1].Status)
		}
		if got.HandlerID == nil || *got.HandlerID != 99 {
			t.Fatalf("handler id not recorded: %+v", got.HandlerID)
		}
		// 已处理的举报不能再次处理。
		if _, err := svc.Handle(ctx, 99, rp.ID, &dto.HandleReportRequest{Action: constants.ReportActionReject}); err == nil {
			t.Fatalf("expected conflict on double handling")
		}
		_ = reports
	})

	t.Run("reject keeps product and records reason", func(t *testing.T) {
		svc, _, products := newReportService()
		rp, _ := svc.Create(ctx, 10, &dto.CreateReportRequest{ProductID: 1, Reason: constants.ReportReasonFakeDescription})
		got, err := svc.Handle(ctx, 99, rp.ID, &dto.HandleReportRequest{Action: constants.ReportActionReject, Remark: "描述与实物相符"})
		if err != nil {
			t.Fatalf("handle: %v", err)
		}
		if got.Status != constants.ReportStatusRejected {
			t.Fatalf("report status = %s, want rejected", got.Status)
		}
		if got.HandleRemark != "描述与实物相符" {
			t.Fatalf("reject reason not recorded: %s", got.HandleRemark)
		}
		if products.products[1].Status != constants.ProductStatusOnSale {
			t.Fatalf("product status = %s, want on_sale", products.products[1].Status)
		}
	})

	tests := []struct {
		name        string
		setup       func(svc *ReportService, reports *fakeReportRepo, products *fakeReportProductRepo) uint
		action      string
		productLeft string
		reportLeft  string
	}{
		{
			name: "sold product refuses take down, both unchanged",
			setup: func(_ *ReportService, reports *fakeReportRepo, _ *fakeReportProductRepo) uint {
				// 举报提交时商品在售，处理前交易完成商品已售出。
				rp := &model.Report{ProductID: 2, ReporterID: 10, Reason: constants.ReportReasonProhibitedItem, Status: constants.ReportStatusPending}
				_ = reports.Create(ctx, rp)
				return rp.ID
			},
			action:      constants.ReportActionTakeDown,
			productLeft: constants.ProductStatusSold,
			reportLeft:  constants.ReportStatusPending,
		},
		{
			name: "seller removed product, both unchanged",
			setup: func(_ *ReportService, reports *fakeReportRepo, _ *fakeReportProductRepo) uint {
				// 举报提交时商品在售，处理前卖家自行下架。
				rp := &model.Report{ProductID: 3, ReporterID: 10, Reason: constants.ReportReasonProhibitedItem, Status: constants.ReportStatusPending}
				_ = reports.Create(ctx, rp)
				return rp.ID
			},
			action:      constants.ReportActionTakeDown,
			productLeft: constants.ProductStatusRemoved,
			reportLeft:  constants.ReportStatusPending,
		},
		{
			name: "another admin handled first, both unchanged",
			setup: func(svc *ReportService, reports *fakeReportRepo, _ *fakeReportProductRepo) uint {
				rp, _ := svc.Create(ctx, 10, &dto.CreateReportRequest{ProductID: 1, Reason: constants.ReportReasonProhibitedItem})
				reports.resolveFail = repository.ErrAlreadyHandled
				return rp.ID
			},
			action:      constants.ReportActionTakeDown,
			productLeft: constants.ProductStatusOnSale,
			reportLeft:  constants.ReportStatusPending,
		},
		{
			name: "product removed by seller concurrently, both unchanged",
			setup: func(svc *ReportService, _ *fakeReportRepo, products *fakeReportProductRepo) uint {
				rp, _ := svc.Create(ctx, 10, &dto.CreateReportRequest{ProductID: 1, Reason: constants.ReportReasonProhibitedItem})
				products.updateFail = util.ErrConflict
				return rp.ID
			},
			action:      constants.ReportActionTakeDown,
			productLeft: constants.ProductStatusOnSale,
			reportLeft:  constants.ReportStatusPending,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, reports, products := newReportService()
			reportID := tt.setup(svc, reports, products)
			if _, err := svc.Handle(ctx, 99, reportID, &dto.HandleReportRequest{Action: tt.action, Remark: "x"}); err == nil {
				t.Fatalf("expected conflict error")
			}
			if products.products[reports.reports[reportID].ProductID].Status != tt.productLeft {
				t.Fatalf("product status = %s, want %s", products.products[reports.reports[reportID].ProductID].Status, tt.productLeft)
			}
			if got := reports.reports[reportID].Status; got != tt.reportLeft {
				t.Fatalf("report status = %s, want %s", got, tt.reportLeft)
			}
		})
	}
}
