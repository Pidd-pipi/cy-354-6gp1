package handler

import (
	"fmt"
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

// ProductReportHandler exposes student and admin product-report endpoints.
type ProductReportHandler struct {
	svc    *service.ProductReportService
	users  *service.UserService
	logger *slog.Logger
}

// NewProductReportHandler wires the product report handler dependencies.
func NewProductReportHandler(svc *service.ProductReportService, users *service.UserService, logger *slog.Logger) *ProductReportHandler {
	return &ProductReportHandler{svc: svc, users: users, logger: logger}
}

// Submit handles POST /reports/products.
func (h *ProductReportHandler) Submit(c *gin.Context) {
	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		util.Fail(c, http.StatusUnauthorized, constants.CodeUnauthorized, constants.MsgUnauthorized)
		return
	}
	var req dto.CreateProductReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(fmt.Sprintf("product_report[product=%d] student submit bind: %v", req.ProductID, err))
		util.Fail(c, http.StatusBadRequest, constants.CodeValidation, constants.MsgValidationFailed)
		return
	}
	user, err := h.users.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	rp, err := h.svc.Submit(c.Request.Context(), user, &req)
	if err != nil {
		c.Error(err)
		return
	}
	if rp.Duplicated {
		util.OKWithMessage(c, "你已对该商品提交过举报，请勿重复提交", rp)
		return
	}
	util.OKWithMessage(c, "举报已提交，等待管理员处理", rp)
}

// ListMine handles GET /reports/products/me.
func (h *ProductReportHandler) ListMine(c *gin.Context) {
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

// ListPending handles GET /admin/reports/products.
func (h *ProductReportHandler) ListPending(c *gin.Context) {
	items, err := h.svc.ListPending(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, items)
}

// Handle handles POST /admin/reports/products/:id/handle.
func (h *ProductReportHandler) Handle(c *gin.Context) {
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
	var req dto.HandleProductReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(fmt.Sprintf("product_report[id=%d] admin handle bind: %v", id, err))
		util.Fail(c, http.StatusBadRequest, constants.CodeValidation, constants.MsgValidationFailed)
		return
	}
	admin, err := h.users.GetProfile(c.Request.Context(), adminID)
	if err != nil {
		c.Error(err)
		return
	}
	rp, err := h.svc.Handle(c.Request.Context(), admin, uint(id), &req)
	if err != nil {
		c.Error(err)
		return
	}
	if req.Action == constants.ReportActionTakeDown {
		util.OKWithMessage(c, "已下架商品并反馈举报结果", rp)
		return
	}
	util.OKWithMessage(c, "已驳回举报，商品保持在售", rp)
}
