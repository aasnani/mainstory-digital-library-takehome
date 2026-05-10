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

type AuthHandler struct {
	cfg *config.Config
	svc *service.UserService
}

func NewAuthHandler(cfg *config.Config, svc *service.UserService) *AuthHandler {
	return &AuthHandler{cfg: cfg, svc: svc}
}

type authReq struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type authResp struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        *domain.User `json:"user"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req authReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid JSON body")
		return
	}
	u, tok, err := h.svc.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
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

func (h *AuthHandler) Login(c *gin.Context) {
	var req authReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid JSON body")
		return
	}
	u, tok, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
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
