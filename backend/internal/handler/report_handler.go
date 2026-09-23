package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
	"github.com/lp/campus-market/internal/middleware"
	"github.com/lp/campus-market/internal/service"
	"github.com/lp/campus-market/internal/util"
)

// ReportHandler exposes product report endpoints for students and admins.
type ReportHandler struct {
	svc    *service.ReportService
	logger *slog.Logger
}

// NewReportHandler wires the report handler dependencies.
func NewReportHandler(svc *service.ReportService, logger *slog.Logger) *ReportHandler {
	return &ReportHandler{svc: svc, logger: logger}
}

// Create handles POST /reports.
func (h *ReportHandler) Create(c *gin.Context) {
	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		util.Fail(c, http.StatusUnauthorized, constants.CodeUnauthorized, constants.MsgUnauthorized)
		return
	}
	var req dto.CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidation, constants.MsgValidationFailed)
		return
	}
	rp, err := h.svc.Create(c.Request.Context(), userID, &req)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, rp)
}

// ListMine handles GET /reports/me.
func (h *ReportHandler) ListMine(c *gin.Context) {
	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		util.Fail(c, http.StatusUnauthorized, constants.CodeUnauthorized, constants.MsgUnauthorized)
		return
	}
	items, err := h.svc.ListMine(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, items)
}

// ListPending handles GET /admin/reports.
func (h *ReportHandler) ListPending(c *gin.Context) {
	var q dto.ListReportQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidation, constants.MsgValidationFailed)
		return
	}
	result, err := h.svc.ListPending(c.Request.Context(), &q)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, result)
}

// Handle handles POST /admin/reports/:id/handle.
func (h *ReportHandler) Handle(c *gin.Context) {
	adminID, err := middleware.CurrentUserID(c)
	if err != nil {
		util.Fail(c, http.StatusUnauthorized, constants.CodeUnauthorized, constants.MsgUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "举报ID不合法")
		return
	}
	var req dto.HandleReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidation, constants.MsgValidationFailed)
		return
	}
	rp, err := h.svc.Handle(c.Request.Context(), adminID, uint(id), &req)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, rp)
}
