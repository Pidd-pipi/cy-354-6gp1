package model

import "time"

// Report is a student complaint against an on-sale product. One student can
// only keep one pending record for the same product.
type Report struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	ProductID    uint       `gorm:"uniqueIndex:uniq_report_product_reporter,priority:1;not null" json:"product_id"`
	ReporterID   uint       `gorm:"uniqueIndex:uniq_report_product_reporter,priority:2;index;not null" json:"reporter_id"`
	Reason       string     `gorm:"size:32;not null" json:"reason"`
	Description  string     `gorm:"type:text" json:"description"`
	Status       string     `gorm:"size:16;index;not null;default:pending" json:"status"`
	HandlerID    *uint      `gorm:"index" json:"handler_id"`
	HandleRemark string     `gorm:"type:text" json:"handle_remark"`
	HandledAt    *time.Time `json:"handled_at"`
	CreatedAt    time.Time  `json:"created_at"`
}
