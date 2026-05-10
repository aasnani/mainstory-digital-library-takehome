package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/api"
	"mainstory-digital-library-takehome/internal/domain"
	"mainstory-digital-library-takehome/internal/middleware"
	"mainstory-digital-library-takehome/internal/service"
)

// UsersHandler exposes profile and admin user management; authorization checks stay here, policy in UserService.
type UsersHandler struct {
	svc *service.UserService
}

// NewUsersHandler constructs the handler with its only dependency (service), keeping Gin out of business logic.
func NewUsersHandler(svc *service.UserService) *UsersHandler {
	return &UsersHandler{svc: svc}
}

// patchUserReq uses pointers so JSON omission means “don’t change” vs empty string for admin updates.
type patchUserReq struct {
	Email           *string `json:"email"`
	Role            *string `json:"role"`
	CurrentPassword *string `json:"current_password"`
	NewPassword     *string `json:"new_password"`
}

// Me returns GET /users/me — the authenticated user row (no password fields).
func (h *UsersHandler) Me(c *gin.Context) {
	id, ok := middleware.UserID(c)
	if !ok {
		api.WriteError(c, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}
	u, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

// PatchMe handles PATCH /users/me: self-service password or (if admin) same routing with admin flag from JWT.
func (h *UsersHandler) PatchMe(c *gin.Context) {
	id, ok := middleware.UserID(c)
	if !ok {
		api.WriteError(c, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}
	// Admins hitting /users/me still act on self only; flag lets service allow email/role when role is ADMIN.
	role, _ := middleware.Role(c)
	var req patchUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid JSON body")
		return
	}
	u, err := h.svc.Patch(c.Request.Context(), id, id, service.PatchInput{
		Email:           req.Email,
		Role:            req.Role,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}, role == domain.RoleAdmin)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

// List is GET /users with limit/offset — admin-only; duplicates pagination parsing pattern from books for consistency.
func (h *UsersHandler) List(c *gin.Context) {
	role, ok := middleware.Role(c)
	if !ok || role != domain.RoleAdmin {
		api.WriteError(c, http.StatusForbidden, "forbidden", "admin role required")
		return
	}
	limit := int32(50)
	offset := int32(0)
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			api.WriteError(c, http.StatusBadRequest, "validation_error", "limit must be between 1 and 100")
			return
		}
		limit = int32(n)
	}
	if v := c.Query("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			api.WriteError(c, http.StatusBadRequest, "validation_error", "offset must be non-negative")
			return
		}
		offset = int32(n)
	}
	users, err := h.svc.List(c.Request.Context(), limit, offset)
	if err != nil {
		api.WriteError(c, http.StatusInternalServerError, "internal_error", "failed to list users")
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

// GetByID returns one user for admin or for self (same id as JWT subject).
func (h *UsersHandler) GetByID(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}
	selfID, ok := middleware.UserID(c)
	if !ok {
		api.WriteError(c, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}
	role, _ := middleware.Role(c)
	if role != domain.RoleAdmin && selfID != targetID {
		api.WriteError(c, http.StatusForbidden, "forbidden", "cannot access other users")
		return
	}
	u, err := h.svc.GetByID(c.Request.Context(), targetID)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

// PatchByID is PATCH /users/:id — delegates admin vs self rules to UserService.Patch via isAdmin.
func (h *UsersHandler) PatchByID(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}
	selfID, ok := middleware.UserID(c)
	if !ok {
		api.WriteError(c, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}
	role, _ := middleware.Role(c)
	isAdmin := role == domain.RoleAdmin
	var req patchUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid JSON body")
		return
	}
	u, err := h.svc.Patch(c.Request.Context(), selfID, targetID, service.PatchInput{
		Email:           req.Email,
		Role:            req.Role,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}, isAdmin)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

// DeleteByID is admin-only DELETE; 409 when entitlements FK block delete.
func (h *UsersHandler) DeleteByID(c *gin.Context) {
	role, ok := middleware.Role(c)
	if !ok || role != domain.RoleAdmin {
		api.WriteError(c, http.StatusForbidden, "forbidden", "admin role required")
		return
	}
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}
	err = h.svc.Delete(c.Request.Context(), targetID)
	if err != nil {
		if errors.Is(err, domain.ErrCannotDeleteUser) {
			api.WriteErrorFromDomain(c, err)
			return
		}
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
