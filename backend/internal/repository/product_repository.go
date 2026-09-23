package repository

import (
	"context"

	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProductRepository persists product rows.
type ProductRepository struct {
	db *gorm.DB
}

// NewProductRepository builds a ProductRepository.
func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// Create inserts a new product.
func (r *ProductRepository) Create(ctx context.Context, p *model.Product) error {
	return db(ctx, r.db).Create(p).Error
}

// FindByID returns a product by id.
func (r *ProductRepository) FindByID(ctx context.Context, id uint) (*model.Product, error) {
	var p model.Product
	err := db(ctx, r.db).First(&p, id).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return &p, nil
}

// List filters products by category/campus/keyword/status with pagination.
func (r *ProductRepository) List(ctx context.Context, category, campus, keyword, status string, page, pageSize int) ([]model.Product, int64, error) {
	q := db(ctx, r.db).Model(&model.Product{})
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if campus != "" {
		q = q.Where("campus = ?", campus)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if keyword != "" {
		q = q.Where("title LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.Product
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// UpdateStatus sets the product status.
func (r *ProductRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	res := db(ctx, r.db).Model(&model.Product{}).Where("id = ?", id).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return util.ErrNotFound
	}
	return nil
}

// UpdateStatusIfOnSale sets the product status only while the product is
// still on sale; it returns ErrConflict when the product has already moved
// out of the on_sale state (sold / removed / reserved by another flow).
func (r *ProductRepository) UpdateStatusIfOnSale(ctx context.Context, id uint, status string) error {
	res := db(ctx, r.db).Model(&model.Product{}).
		Where("id = ? AND status = ?", id, "on_sale").
		Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return util.ErrConflict
	}
	return nil
}

// FindByIDForUpdate returns the product row with a row lock, usable only
// inside a transaction.
func (r *ProductRepository) FindByIDForUpdate(ctx context.Context, id uint) (*model.Product, error) {
	var p model.Product
	err := db(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, id).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return &p, nil
}

// FindByIDs returns products matching the given ids.
func (r *ProductRepository) FindByIDs(ctx context.Context, ids []uint) ([]model.Product, error) {
	if len(ids) == 0 {
		return []model.Product{}, nil
	}
	var items []model.Product
	err := db(ctx, r.db).Where("id IN ?", ids).Find(&items).Error
	return items, err
}

// Count returns the total product count.
func (r *ProductRepository) Count(ctx context.Context) (int64, error) {
	var n int64
	err := db(ctx, r.db).Model(&model.Product{}).Count(&n).Error
	return n, err
}
