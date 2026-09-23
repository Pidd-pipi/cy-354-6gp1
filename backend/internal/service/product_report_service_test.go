package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
)

// fakeReportProductRepo is an in-memory ProductRepository stand-in supporting
// the report service test paths.
type fakeReportProductRepo struct {
	mu       sync.Mutex
	products map[uint]*model.Product
}

func newFakeReportProductRepo() *fakeReportProductRepo {
	return &fakeReportProductRepo{products: map[uint]*model.Product{}}
}

func (f *fakeReportProductRepo) seed(p *model.Product) { f.products[p.ID] = p }

func (f *fakeReportProductRepo) Create(_ context.Context, p *model.Product) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.products[p.ID] = p
	return nil
}

func (f *fakeReportProductRepo) FindByID(ctx context.Context, id uint) (*model.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if p, ok := f.products[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, util.ErrNotFound
}

func (f *fakeReportProductRepo) FindByIDForUpdate(ctx context.Context, id uint) (*model.Product, error) {
	return f.FindByID(ctx, id)
}

func (f *fakeReportProductRepo) FindByIDs(_ context.Context, ids []uint) ([]model.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]model.Product, 0, len(ids))
	for _, id := range ids {
		if p, ok := f.products[id]; ok {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (f *fakeReportProductRepo) UpdateStatus(_ context.Context, id uint, status string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.products[id]
	if !ok {
		return util.ErrNotFound
	}
	p.Status = status
	return nil
}

func (f *fakeReportProductRepo) UpdateStatusIfOnSale(_ context.Context, id uint, status string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
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

// fakeReportUserRepo satisfies the user lookup surface used by report views.
type fakeReportUserRepo struct {
	users map[uint]*model.User
}

func newFakeReportUserRepo() *fakeReportUserRepo {
	return &fakeReportUserRepo{users: map[uint]*model.User{}}
}

func (f *fakeReportUserRepo) FindByID(_ context.Context, id uint) (*model.User, error) {
	if u, ok := f.users[id]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, util.ErrNotFound
}

func (f *fakeReportUserRepo) FindByIDs(_ context.Context, ids []uint) ([]model.User, error) {
	out := make([]model.User, 0, len(ids))
	for _, id := range ids {
		if u, ok := f.users[id]; ok {
			out = append(out, *u)
		}
	}
	return out, nil
}

// fakeProductReportStore is an in-memory ProductReportRepository stand-in.
type fakeProductReportStore struct {
	mu     sync.Mutex
	rows   map[uint]*model.ProductReport
	nextID uint
	dedup  map[string]uint
}

func newFakeProductReportStore() *fakeProductReportStore {
	return &fakeProductReportStore{rows: map[uint]*model.ProductReport{}, nextID: 1, dedup: map[string]uint{}}
}

func (s *fakeProductReportStore) Create(_ context.Context, rp *model.ProductReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rp.DedupKey != nil {
		if _, exists := s.dedup[*rp.DedupKey]; exists {
			return errors.New("Error 1062: Duplicate entry for key 'uniq_product_reports_dedup'")
		}
	}
	rp.ID = s.nextID
	s.nextID++
	cp := *rp
	s.rows[rp.ID] = &cp
	if rp.DedupKey != nil {
		s.dedup[*rp.DedupKey] = rp.ID
	}
	return nil
}

func (s *fakeProductReportStore) getLocked(id uint) (*model.ProductReport, error) {
	rp, ok := s.rows[id]
	if !ok {
		return nil, util.ErrNotFound
	}
	cp := *rp
	return &cp, nil
}

func (s *fakeProductReportStore) FindByID(_ context.Context, id uint) (*model.ProductReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(id)
}

func (s *fakeProductReportStore) FindByIDForUpdate(ctx context.Context, id uint) (*model.ProductReport, error) {
	return s.FindByID(ctx, id)
}

func (s *fakeProductReportStore) FindPendingByProductAndReporter(_ context.Context, productID, reporterID uint) (*model.ProductReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rp := range s.rows {
		if rp.ProductID == productID && rp.ReporterID == reporterID && rp.Status == constants.ReportStatusPending {
			cp := *rp
			return &cp, nil
		}
	}
	return nil, util.ErrNotFound
}

func (s *fakeProductReportStore) ListByReporter(_ context.Context, reporterID uint) ([]model.ProductReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []model.ProductReport{}
	for _, rp := range s.rows {
		if rp.ReporterID == reporterID {
			out = append(out, *rp)
		}
	}
	return out, nil
}

func (s *fakeProductReportStore) ListByStatus(_ context.Context, status string) ([]model.ProductReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []model.ProductReport{}
	for _, rp := range s.rows {
		if rp.Status == status {
			out = append(out, *rp)
		}
	}
	return out, nil
}

func (s *fakeProductReportStore) Complete(_ context.Context, id, handlerID uint, status, remark string, ts interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rp, ok := s.rows[id]
	if !ok {
		return util.ErrNotFound
	}
	if rp.Status != constants.ReportStatusPending {
		return util.ErrConflict
	}
	rp.Status = status
	rp.HandlerID = &handlerID
	rp.HandleRemark = remark
	if t, ok := ts.(time.Time); ok {
		rp.HandledAt = &t
	}
	if rp.DedupKey != nil {
		delete(s.dedup, *rp.DedupKey)
		rp.DedupKey = nil
	}
	return nil
}

func (s *fakeProductReportStore) Transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}

func newReportServiceHarness() (*ProductReportService, *fakeReportProductRepo, *fakeProductReportStore) {
	products := newFakeReportProductRepo()
	users := newFakeReportUserRepo()
	users.users[1] = &model.User{ID: 1, Nickname: "小明同学"}
	users.users[2] = &model.User{ID: 2, Nickname: "阿珍"}
	users.users[9] = &model.User{ID: 9, Nickname: "管理员", Role: constants.UserRoleAdmin}
	products.seed(&model.Product{ID: 10, SellerID: 1, Title: "在售商品", Status: constants.ProductStatusOnSale})
	products.seed(&model.Product{ID: 11, SellerID: 1, Title: "已售出商品", Status: constants.ProductStatusSold})
	reports := newFakeProductReportStore()
	svc := NewProductReportService(reports, products, users, slog.Default())
	return svc, products, reports
}

func TestProductReportSubmitDuplicateReturnsOriginal(t *testing.T) {
	svc, _, _ := newReportServiceHarness()
	reporter := &model.User{ID: 2}
	req := &dto.CreateProductReportRequest{ProductID: 10, Reason: constants.ReportReasonFalseDescription, Description: "描述与实物不符"}
	first, err := svc.Submit(context.Background(), reporter, req)
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	if first.Duplicated {
		t.Fatalf("first submit must not be flagged duplicated")
	}
	second, err := svc.Submit(context.Background(), reporter, req)
	if err != nil {
		t.Fatalf("duplicate submit must return original, got error: %v", err)
	}
	if !second.Duplicated || second.ID != first.ID {
		t.Fatalf("expected original pending record id=%d duplicated=true, got id=%d duplicated=%v", first.ID, second.ID, second.Duplicated)
	}
}

func TestProductReportSubmitGuards(t *testing.T) {
	svc, _, _ := newReportServiceHarness()
	tests := []struct {
		name     string
		reporter *model.User
		req      *dto.CreateProductReportRequest
		wantCode int
	}{
		{name: "invalid reason", reporter: &model.User{ID: 2}, req: &dto.CreateProductReportRequest{ProductID: 10, Reason: "nope"}, wantCode: constants.CodeValidation},
		{name: "self report", reporter: &model.User{ID: 1}, req: &dto.CreateProductReportRequest{ProductID: 10, Reason: constants.ReportReasonOther}, wantCode: constants.CodeBadRequest},
		{name: "missing product", reporter: &model.User{ID: 2}, req: &dto.CreateProductReportRequest{ProductID: 99, Reason: constants.ReportReasonOther}, wantCode: constants.CodeNotFound},
		{name: "sold product", reporter: &model.User{ID: 2}, req: &dto.CreateProductReportRequest{ProductID: 11, Reason: constants.ReportReasonProhibitedItem}, wantCode: constants.CodeConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Submit(context.Background(), tt.reporter, tt.req)
			var appErr *util.AppError
			if !errors.As(err, &appErr) || appErr.Code != tt.wantCode {
				t.Fatalf("want code %d, got %v", tt.wantCode, err)
			}
		})
	}
}

func TestProductReportHandleTakeDownAtomic(t *testing.T) {
	svc, products, reports := newReportServiceHarness()
	reporter := &model.User{ID: 2}
	created, err := svc.Submit(context.Background(), reporter, &dto.CreateProductReportRequest{ProductID: 10, Reason: constants.ReportReasonProhibitedItem})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	admin := &model.User{ID: 9, Role: constants.UserRoleAdmin}
	view, err := svc.Handle(context.Background(), admin, created.ID, &dto.HandleProductReportRequest{Action: constants.ReportActionTakeDown})
	if err != nil {
		t.Fatalf("take down: %v", err)
	}
	if view.Status != constants.ReportStatusTakenDown || products.products[10].Status != constants.ProductStatusRemoved {
		t.Fatalf("expected report taken_down and product removed")
	}
	stored := reports.rows[created.ID]
	if stored.Status != constants.ReportStatusTakenDown || stored.HandlerID == nil || *stored.HandlerID != 9 || stored.DedupKey != nil {
		t.Fatalf("handled report row mismatch: %+v", stored)
	}
	// second admin handling must fail with both sides unchanged
	_, err = svc.Handle(context.Background(), admin, created.ID, &dto.HandleProductReportRequest{Action: constants.ReportActionTakeDown})
	if !errors.Is(err, util.ErrConflict) {
		var appErr *util.AppError
		if !errors.As(err, &appErr) || appErr.Code != constants.CodeConflict {
			t.Fatalf("expected conflict on double handle, got %v", err)
		}
	}
}

func TestProductReportTakeDownRejectedWhenProductGone(t *testing.T) {
	svc, products, _ := newReportServiceHarness()
	created, err := svc.Submit(context.Background(), &model.User{ID: 2}, &dto.CreateProductReportRequest{ProductID: 10, Reason: constants.ReportReasonProhibitedItem})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	// product sold by a normal trade flow before the admin handles the report
	products.products[10].Status = constants.ProductStatusSold
	_, err = svc.Handle(context.Background(), &model.User{ID: 9}, created.ID, &dto.HandleProductReportRequest{Action: constants.ReportActionTakeDown})
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Code != constants.CodeConflict {
		t.Fatalf("expected conflict for gone product, got %v", err)
	}
}

func TestProductReportRejectKeepsProductAndNeedsRemark(t *testing.T) {
	svc, products, _ := newReportServiceHarness()
	created, err := svc.Submit(context.Background(), &model.User{ID: 2}, &dto.CreateProductReportRequest{ProductID: 10, Reason: constants.ReportReasonFalseDescription})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	_, err = svc.Handle(context.Background(), &model.User{ID: 9}, created.ID, &dto.HandleProductReportRequest{Action: constants.ReportActionReject})
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Code != constants.CodeValidation {
		t.Fatalf("reject without remark must be validation error, got %v", err)
	}
	view, err := svc.Handle(context.Background(), &model.User{ID: 9}, created.ID, &dto.HandleProductReportRequest{Action: constants.ReportActionReject, Remark: "描述属实"})
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if view.Status != constants.ReportStatusRejected || products.products[10].Status != constants.ProductStatusOnSale {
		t.Fatalf("rejected report must keep product on sale")
	}
}

func TestProductReportDuplicateAfterProductGoneReturnsOriginal(t *testing.T) {
	svc, products, _ := newReportServiceHarness()
	reporter := &model.User{ID: 2}
	req := &dto.CreateProductReportRequest{ProductID: 10, Reason: constants.ReportReasonOther}
	first, err := svc.Submit(context.Background(), reporter, req)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	products.products[10].Status = constants.ProductStatusSold
	again, err := svc.Submit(context.Background(), reporter, req)
	if err != nil {
		t.Fatalf("duplicate submit must still return original: %v", err)
	}
	if !again.Duplicated || again.ID != first.ID {
		t.Fatalf("expected original pending record id=%d, got id=%d", first.ID, again.ID)
	}
}

func TestProductReportCanResubmitAfterRejection(t *testing.T) {
	svc, _, _ := newReportServiceHarness()
	reporter := &model.User{ID: 2}
	first, err := svc.Submit(context.Background(), reporter, &dto.CreateProductReportRequest{ProductID: 10, Reason: constants.ReportReasonOther})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := svc.Handle(context.Background(), &model.User{ID: 9}, first.ID, &dto.HandleProductReportRequest{Action: constants.ReportActionReject, Remark: "不成立"}); err != nil {
		t.Fatalf("reject: %v", err)
	}
	second, err := svc.Submit(context.Background(), reporter, &dto.CreateProductReportRequest{ProductID: 10, Reason: constants.ReportReasonFraud})
	if err != nil {
		t.Fatalf("resubmit after rejection must be allowed: %v", err)
	}
	if second.Duplicated || second.ID == first.ID {
		t.Fatalf("expected a fresh pending report after rejection, got duplicated=%v id=%d", second.Duplicated, second.ID)
	}
}
