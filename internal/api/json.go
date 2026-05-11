// Package api centralizes JSON error envelopes so every handler returns the same shape the SPA documents.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mainstory-digital-library-takehome/internal/domain"
)

// ErrorBody is the inner object so clients can branch on machine-readable code while showing message to users.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse wraps ErrorBody under "error" to match docs/api-contract.md and many OAuth-style APIs.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// WriteError writes an arbitrary validation or infrastructure error with explicit status and code.
func WriteError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
}

// WriteErrorFromDomain maps sentinel domain errors to stable HTTP codes/messages (contract in docs/api-contract.md).
func WriteErrorFromDomain(c *gin.Context, err error) {
	switch err {
	case domain.ErrInvalidEmail:
		WriteError(c, http.StatusBadRequest, "validation_error", "invalid email")
	case domain.ErrInvalidRole:
		WriteError(c, http.StatusBadRequest, "validation_error", "invalid role")
	case domain.ErrInvalidPassword:
		WriteError(c, http.StatusBadRequest, "validation_error", "password must be 8–72 characters")
	case domain.ErrInvalidPasswordChange:
		WriteError(c, http.StatusBadRequest, "validation_error", "current_password and new_password must both be provided")
	case domain.ErrEmptyPatch:
		WriteError(c, http.StatusBadRequest, "validation_error", "no fields to update")
	case domain.ErrCannotPatchOtherUserPassword:
		WriteError(c, http.StatusBadRequest, "validation_error", "password can only be changed on your own account")
	case domain.ErrNotFound:
		WriteError(c, http.StatusNotFound, "not_found", "resource not found")
	case domain.ErrConflict:
		WriteError(c, http.StatusConflict, "conflict", "conflict")
	case domain.ErrForbidden:
		WriteError(c, http.StatusForbidden, "forbidden", "forbidden")
	case domain.ErrUnauthorized:
		WriteError(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
	case domain.ErrCannotDeleteUser:
		WriteError(c, http.StatusConflict, "conflict", "cannot delete user with existing entitlements")
	case domain.ErrInvalidBook:
		WriteError(c, http.StatusBadRequest, "validation_error", "invalid book")
	case domain.ErrSearchTermTooShort:
		WriteError(c, http.StatusBadRequest, "validation_error", "use at least 2 characters for search filters (q, title, author, or user list q)")
	case domain.ErrInvalidCatalogFilters:
		WriteError(c, http.StatusBadRequest, "validation_error", "invalid catalog filters (check price range)")
	case domain.ErrInvalidPrice:
		WriteError(c, http.StatusBadRequest, "validation_error", "invalid price")
	case domain.ErrInvalidEntitlementType:
		WriteError(c, http.StatusBadRequest, "validation_error", "invalid entitlement type")
	case domain.ErrInvalidEntitlementStatus:
		WriteError(c, http.StatusBadRequest, "validation_error", "invalid entitlement status")
	case domain.ErrInvalidEntitlementShape:
		WriteError(c, http.StatusBadRequest, "validation_error", "invalid entitlement fields for type")
	case domain.ErrInvalidEntitlementRequest:
		WriteError(c, http.StatusBadRequest, "validation_error", "user_id required when admin creates an entitlement")
	case domain.ErrNoActiveSubscription:
		WriteError(c, http.StatusNotFound, "no_active_subscription", "no active subscription to cancel")
	default:
		// WHAT: non-sentinel errors become 500 — avoids leaking implementation details while staying JSON-shaped.
		WriteError(c, http.StatusInternalServerError, "internal_error", "internal error")
	}
}
