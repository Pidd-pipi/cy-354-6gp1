package dto

import "time"

// CreateProductReportRequest is the payload a student submits against an on-sale product.
type CreateProductReportRequest struct {
	ProductID   uint   `json:"product_id" binding:"required"`
	Reason      string `json:"reason" binding:"required,oneof=false_description prohibited_item fraud other"`
	Description string `json:"description" binding:"max=500"`
}

// HandleProductReportRequest is the admin payload for handling a pending report.
type HandleProductReportRequest struct {
	Action string `json:"action" binding:"required,oneof=take_down reject"`
	Remark string `json:"remark" binding:"max=500"`
}

// ProductReportView is the enriched report payload shared with the frontend
// (report rows plus product/reporter display fields).
type ProductReportView struct {
	ID            uint       `json:"id"`
	ProductID     uint       `json:"product_id"`
	ReporterID    uint       `json:"reporter_id"`
	Reason        string     `json:"reason"`
	Description   string     `json:"description"`
	Status        string     `json:"status"`
	HandlerID     *uint      `json:"handler_id"`
	HandleRemark  string     `json:"handle_remark"`
	HandledAt     *time.Time `json:"handled_at"`
	CreatedAt     time.Time  `json:"created_at"`
	ProductTitle  string     `json:"product_title"`
	ProductStatus string     `json:"product_status"`
	ReporterName  string     `json:"reporter_name"`
	Duplicated    bool       `json:"duplicated"`
}
