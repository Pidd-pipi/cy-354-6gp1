package repository

import (
	"context"
	"strings"

	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProductReportRepository persists product report rows.
type ProductReportRepository struct {
	db *gorm.DB
}

// NewProductReportRepository builds a ProductReportRepository.
func NewProductReportRepository(db *gorm.DB) *ProductReportRepository {
	return &ProductReportRepository{db: db}
}

// Transaction runs fn inside a database transaction for cross-repository writes.
func (r *ProductReportRepository) Transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return Transaction(ctx, r.db, fn)
}

// Create inserts a new product report.
func (r *ProductReportRepository) Create(ctx context.Context, rp *model.ProductReport) error {
	return db(ctx, r.db).Create(rp).Error
}

// FindByID returns a product report by id.
func (r *ProductReportRepository) FindByID(ctx context.Context, id uint) (*model.ProductReport, error) {
	var rp model.ProductReport
	err := db(ctx, r.db).First(&rp, id).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return &rp, nil
}

// FindByIDForUpdate returns the report row with a row lock, usable only
// inside a transaction.
func (r *ProductReportRepository) FindByIDForUpdate(ctx context.Context, id uint) (*model.ProductReport, error) {
	var rp model.ProductReport
	err := db(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).First(&rp, id).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return &rp, nil
}

// FindPendingByProductAndReporter returns the pending report of a reporter
// for a product, or util.ErrNotFound when none exists.
func (r *ProductReportRepository) FindPendingByProductAndReporter(ctx context.Context, productID, reporterID uint) (*model.ProductReport, error) {
	var rp model.ProductReport
	err := db(ctx, r.db).
		Where("product_id = ? AND reporter_id = ? AND status = ?", productID, reporterID, "pending").
		First(&rp).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return &rp, nil
}

// ListByReporter returns the reports submitted by a user.
func (r *ProductReportRepository) ListByReporter(ctx context.Context, reporterID uint) ([]model.ProductReport, error) {
	var items []model.ProductReport
	err := db(ctx, r.db).Where("reporter_id = ?", reporterID).
		Order("created_at DESC").Find(&items).Error
	return items, err
}

// ListByStatus returns reports in the given status, newest first.
func (r *ProductReportRepository) ListByStatus(ctx context.Context, status string) ([]model.ProductReport, error) {
	var items []model.ProductReport
	err := db(ctx, r.db).Where("status = ?", status).
		Order("created_at DESC").Find(&items).Error
	return items, err
}

// Complete moves a pending report to its final status atomically, clears the
// pending dedup key and records the handler. It returns util.ErrConflict when
// the report was already handled by another admin.
func (r *ProductReportRepository) Complete(ctx context.Context, id, handlerID uint, status, remark string, ts interface{}) error {
	res := db(ctx, r.db).Model(&model.ProductReport{}).
		Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]interface{}{
			"status":        status,
			"handler_id":    handlerID,
			"handle_remark": remark,
			"handled_at":    ts,
			"dedup_key":     nil,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return util.ErrConflict
	}
	return nil
}

// IsDuplicateKeyError reports whether err is a unique-constraint violation on
// the pending dedup key, meaning another in-flight request just inserted the
// same pending report.
func IsDuplicateKeyError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Duplicate entry")
}
