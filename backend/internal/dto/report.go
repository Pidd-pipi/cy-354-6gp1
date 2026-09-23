package dto

// CreateReportRequest is the payload for a student reporting a product.
type CreateReportRequest struct {
	ProductID   uint   `json:"product_id" binding:"required"`
	Reason      string `json:"reason" binding:"required,oneof=fake_description prohibited_item"`
	Description string `json:"description" binding:"max=500"`
}

// HandleReportRequest is the payload for an admin resolving a report.
type HandleReportRequest struct {
	Action string `json:"action" binding:"required,oneof=take_down reject"`
	Remark string `json:"remark" binding:"max=500"`
}

// ListReportQuery adds a status filter to pagination (admin pending list).
type ListReportQuery struct {
	PageQuery
	Status string `form:"status"`
}
