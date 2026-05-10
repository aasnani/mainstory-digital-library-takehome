package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/api"
	"mainstory-digital-library-takehome/internal/middleware"
	"mainstory-digital-library-takehome/internal/service"
)

type EntitlementsHandler struct {
	svc *service.EntitlementService
}

func NewEntitlementsHandler(svc *service.EntitlementService) *EntitlementsHandler {
	return &EntitlementsHandler{svc: svc}
}

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

type createEntitlementReq struct {
	UserID *uuid.UUID `json:"user_id"`
	Type   string     `json:"type" binding:"required"`
	BookID *uuid.UUID `json:"book_id"`
	Status string     `json:"status"`
}

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

type patchEntitlementReq struct {
	Status *string    `json:"status"`
	EndsAt *time.Time `json:"ends_at"`
}

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
