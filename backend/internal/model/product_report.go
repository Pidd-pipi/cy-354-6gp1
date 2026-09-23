package model

import "time"

// ProductReport is a student report against an on-sale product (false
// description or prohibited item). A reporter keeps a single pending row per
// product; DedupKey holds the "productID:reporterID" pair while the report is
// pending and is cleared once an admin handles it, so the pair can report again.
type ProductReport struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	ProductID    uint       `gorm:"index;not null" json:"product_id"`
	ReporterID   uint       `gorm:"index;not null" json:"reporter_id"`
	Reason       string     `gorm:"size:24;not null" json:"reason"`
	Description  string     `gorm:"type:text" json:"description"`
	Status       string     `gorm:"size:16;index;not null;default:pending" json:"status"`
	HandlerID    *uint      `gorm:"index" json:"handler_id"`
	HandleRemark string     `gorm:"type:text" json:"handle_remark"`
	HandledAt    *time.Time `json:"handled_at"`
	DedupKey     *string    `gorm:"size:48;uniqueIndex" json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
}
