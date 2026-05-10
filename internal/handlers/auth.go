// Package handlers are Gin adapters: parse JSON, call services, map errors—no SQL or entitlement rules here.
package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"mainstory-digital-library-takehome/internal/api"
	"mainstory-digital-library-takehome/internal/config"
	"mainstory-digital-library-takehome/internal/domain"
	"mainstory-digital-library-takehome/internal/service"
)

// AuthHandler needs Config in addition to UserService to echo JWT expiry seconds consistent with the signed token.
type AuthHandler struct {
	cfg *config.Config
	svc *service.UserService
}

func NewAuthHandler(cfg *config.Config, svc *service.UserService) *AuthHandler {
	return &AuthHandler{cfg: cfg, svc: svc}
}

// authReq is register/login JSON — both flows share validation and service calls.
type authReq struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// authResp follows OAuth2-ish field names so frontends can reuse generic Bearer client helpers.
type authResp struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        *domain.User `json:"user"`
}

// Register creates MEMBER + returns JWT in one round-trip so the SPA can store the token immediately.
func (h *AuthHandler) Register(c *gin.Context) {
	var req authReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid JSON body")
		return
	}
	u, tok, err := h.svc.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		// WHAT: surface duplicate email as 409 with explicit message (unique index becomes user-friendly).
		if errors.Is(err, domain.ErrConflict) {
			api.WriteError(c, http.StatusConflict, "conflict", "email already registered")
			return
		}
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusCreated, authResp{
		AccessToken: tok,
		TokenType:   "Bearer",
		ExpiresIn:   int64(h.cfg.JWTExpiry.Seconds()),
		User:        u,
	})
}

// Login returns the same JSON shape as Register; wrong credentials always map to one message (no user enumeration).
func (h *AuthHandler) Login(c *gin.Context) {
	var req authReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid JSON body")
		return
	}
	u, tok, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		// WHAT: generic 401 text for both unknown email and bad password by design.
		if errors.Is(err, domain.ErrUnauthorized) {
			api.WriteError(c, http.StatusUnauthorized, "unauthorized", "invalid email or password")
			return
		}
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, authResp{
		AccessToken: tok,
		TokenType:   "Bearer",
		ExpiresIn:   int64(h.cfg.JWTExpiry.Seconds()),
		User:        u,
	})
}
