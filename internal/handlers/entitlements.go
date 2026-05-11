package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/api"
	"mainstory-digital-library-takehome/internal/domain"
	"mainstory-digital-library-takehome/internal/middleware"
	"mainstory-digital-library-takehome/internal/service"
)

// EntitlementsHandler implements mock purchases/subscribe HTTP; real payment processors would sit behind this API later.
type EntitlementsHandler struct {
	svc *service.EntitlementService
}

func NewEntitlementsHandler(svc *service.EntitlementService) *EntitlementsHandler {
	return &EntitlementsHandler{svc: svc}
}

// List returns entitlements: members see only their rows; staff see global history for support tooling.
func (h *EntitlementsHandler) List(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		api.WriteError(c, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}
	role, _ := middleware.Role(c)
	limit, offset, ok := parseLimitOffset(c)
	if !ok {
		return
	}
	items, err := h.svc.List(c.Request.Context(), uid, role, limit, offset)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entitlements": items})
}

// ListStaff is GET /entitlements/staff — librarian/admin browse with optional filters (same limit/offset rules as list).
func (h *EntitlementsHandler) ListStaff(c *gin.Context) {
	limit, offset, ok := parseLimitOffset(c)
	if !ok {
		return
	}
	filter, err := parseEntitlementStaffFilter(c)
	if err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	items, err := h.svc.ListStaff(c.Request.Context(), filter, limit, offset)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entitlements": items})
}

func parseEntitlementStaffFilter(c *gin.Context) (domain.EntitlementListFilter, error) {
	var f domain.EntitlementListFilter
	f.Type = strings.TrimSpace(c.Query("type"))
	f.Status = strings.TrimSpace(c.Query("status"))
	if v := strings.TrimSpace(c.Query("user_id")); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return f, fmt.Errorf("invalid user_id (expected UUID)")
		}
		f.UserID = &id
	}
	if v := strings.TrimSpace(c.Query("book_id")); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return f, fmt.Errorf("invalid book_id (expected UUID)")
		}
		f.BookID = &id
	}
	return f, nil
}

// GetByID loads one entitlement if the caller owns it or has staff role.
func (h *EntitlementsHandler) GetByID(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		api.WriteError(c, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}
	role, _ := middleware.Role(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid entitlement id")
		return
	}
	e, err := h.svc.Get(c.Request.Context(), uid, role, id)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, e)
}

// createEntitlementReq mirrors POST bodies: user_id only for admin grants; members omit it to prevent self→other escalation.
type createEntitlementReq struct {
	UserID *uuid.UUID `json:"user_id"`
	Type   string     `json:"type" binding:"required"`
	BookID *uuid.UUID `json:"book_id"`
	Status string     `json:"status"`
}

// Create is POST /entitlements — member self-checkout or admin grant depending on body user_id and role.
func (h *EntitlementsHandler) Create(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		api.WriteError(c, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}
	role, _ := middleware.Role(c)
	var req createEntitlementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid JSON body")
		return
	}
	e, err := h.svc.Create(c.Request.Context(), uid, role, service.CreateEntitlementInput{
		TargetUserID: req.UserID,
		Type:         req.Type,
		BookID:       req.BookID,
		Status:       req.Status,
	})
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusCreated, e)
}

// patchEntitlementReq is admin PATCH body — optional fields mean “leave unchanged” when nil.
type patchEntitlementReq struct {
	Status *string    `json:"status"`
	EndsAt *time.Time `json:"ends_at"`
}

// CancelMySubscription is member self-service: sets cancelled_at, keeps access until ends_at.
func (h *EntitlementsHandler) CancelMySubscription(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		api.WriteError(c, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}
	e, err := h.svc.CancelMySubscription(c.Request.Context(), uid)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, e)
}

// Patch is admin-only route in main — force status/ends_at for support without member cancel semantics.
func (h *EntitlementsHandler) Patch(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid entitlement id")
		return
	}
	var req patchEntitlementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid JSON body")
		return
	}
	e, err := h.svc.Patch(c.Request.Context(), id, req.Status, req.EndsAt)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, e)
}
