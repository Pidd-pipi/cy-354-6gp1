package router

import (
	"github.com/gin-gonic/gin"
	"github.com/lp/campus-market/internal/handler"
)

// RegisterReportRoutes registers student report endpoints.
func RegisterReportRoutes(g *gin.RouterGroup, h *handler.ReportHandler, auth, apiLimiter gin.HandlerFunc) {
	reports := g.Group("/reports", auth)
	{
		reports.POST("", apiLimiter, h.Create)
		reports.GET("/me", apiLimiter, h.ListMine)
	}
}

// RegisterAdminReportRoutes registers admin-only report management endpoints.
func RegisterAdminReportRoutes(g *gin.RouterGroup, h *handler.ReportHandler, auth, requireAdmin, apiLimiter gin.HandlerFunc) {
	admin := g.Group("/admin/reports", auth, requireAdmin)
	{
		admin.GET("", apiLimiter, h.ListPending)
		admin.POST("/:id/handle", apiLimiter, h.Handle)
	}
}
