package repository

import (
	"context"
	"errors"

	"github.com/lp/campus-market/internal/model"
	"gorm.io/gorm"
)

// ErrAlreadyHandled means a pending report was already resolved by another admin.
var ErrAlreadyHandled = errors.New("report already handled")

// ReportRepository persists product report rows.
type ReportRepository struct {
	db *gorm.DB
}

// NewReportRepository builds a ReportRepository.
func NewReportRepository(db *gorm.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

// Transaction runs fn inside a database transaction for cross-repository writes.
func (r *ReportRepository) Transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return Transaction(ctx, r.db, fn)
}

// Create inserts a new report.
func (r *ReportRepository) Create(ctx context.Context, rp *model.Report) error {
	return db(ctx, r.db).Create(rp).Error
}

// FindByID returns a report by id.
func (r *ReportRepository) FindByID(ctx context.Context, id uint) (*model.Report, error) {
	var rp model.Report
	err := db(ctx, r.db).First(&rp, id).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return &rp, nil
}

// FindByProductAndReporter returns the report a user left for a product.
func (r *ReportRepository) FindByProductAndReporter(ctx context.Context, productID, reporterID uint) (*model.Report, error) {
	var rp model.Report
	err := db(ctx, r.db).
		Where("product_id = ? AND reporter_id = ?", productID, reporterID).
		First(&rp).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return &rp, nil
}

// ListByReporter returns reports submitted by a student, newest first.
func (r *ReportRepository) ListByReporter(ctx context.Context, reporterID uint) ([]model.Report, error) {
	var items []model.Report
	err := db(ctx, r.db).Where("reporter_id = ?", reporterID).
		Order("created_at DESC").Find(&items).Error
	return items, err
}

// ListByStatus returns reports of a given status with pagination, newest first.
func (r *ReportRepository) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]model.Report, int64, error) {
	q := db(ctx, r.db).Model(&model.Report{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.Report
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Resolve atomically finalizes a pending report. RowsAffected == 0 means the
// report was already handled by another admin (lost the race).
func (r *ReportRepository) Resolve(ctx context.Context, id, handlerID uint, status, remark string, handledAt interface{}) error {
	res := db(ctx, r.db).Model(&model.Report{}).
		Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]interface{}{
			"status":        status,
			"handler_id":    handlerID,
			"handle_remark": remark,
			"handled_at":    handledAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAlreadyHandled
	}
	return nil
}
