package router

import (
	"github.com/gin-gonic/gin"
	"github.com/lp/campus-market/internal/handler"
)

// RegisterProductReportRoutes registers the student-facing product report
// endpoints (submit and view own results).
func RegisterProductReportRoutes(g *gin.RouterGroup, h *handler.ProductReportHandler, auth, apiLimiter gin.HandlerFunc) {
	reports := g.Group("/reports/products", auth)
	{
		reports.POST("", apiLimiter, h.Submit)
		reports.GET("/me", apiLimiter, h.ListMine)
	}
}

// RegisterAdminProductReportRoutes registers the admin-only report queue and
// handling endpoints under the admin group.
func RegisterAdminProductReportRoutes(admin *gin.RouterGroup, h *handler.ProductReportHandler, apiLimiter gin.HandlerFunc) {
	admin.GET("/reports/products", apiLimiter, h.ListPending)
	admin.POST("/reports/products/:id/handle", apiLimiter, h.Handle)
}
