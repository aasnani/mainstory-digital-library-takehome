package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mainstory-digital-library-takehome/internal/domain"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

func WriteError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
}

func WriteErrorFromDomain(c *gin.Context, err error) {
	switch err {
	case domain.ErrInvalidEmail:
		WriteError(c, http.StatusBadRequest, "validation_error", "invalid email")
	case domain.ErrInvalidRole:
		WriteError(c, http.StatusBadRequest, "validation_error", "invalid role")
	case domain.ErrInvalidPassword:
		WriteError(c, http.StatusBadRequest, "validation_error", "password must be 8–72 characters")
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
	default:
		WriteError(c, http.StatusInternalServerError, "internal_error", "internal error")
	}
}
