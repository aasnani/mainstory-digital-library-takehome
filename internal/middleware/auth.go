// Package middleware implements cross-cutting HTTP concerns (auth identity on context, CORS) without importing handlers.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/api"
	"mainstory-digital-library-takehome/internal/auth"
	"mainstory-digital-library-takehome/internal/config"
	"mainstory-digital-library-takehome/internal/domain"
)

const (
	ContextUserIDKey = "auth_user_id"
	ContextRoleKey   = "auth_role"
)

// BearerAuth is strict: every protected route needs a valid JWT so services can trust context user id/role.
func BearerAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" {
			api.WriteError(c, http.StatusUnauthorized, "unauthorized", "missing Authorization header")
			c.Abort()
			return
		}
		const prefix = "Bearer "
		if !strings.HasPrefix(h, prefix) {
			api.WriteError(c, http.StatusUnauthorized, "unauthorized", "Authorization must be Bearer token")
			c.Abort()
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(h, prefix))
		if raw == "" {
			api.WriteError(c, http.StatusUnauthorized, "unauthorized", "empty bearer token")
			c.Abort()
			return
		}
		claims, err := auth.Parse(cfg, raw)
		if err != nil {
			api.WriteError(c, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			c.Abort()
			return
		}
		uid, err := auth.UserID(claims)
		if err != nil {
			api.WriteError(c, http.StatusUnauthorized, "unauthorized", "invalid token subject")
			c.Abort()
			return
		}
		c.Set(ContextUserIDKey, uid)
		c.Set(ContextRoleKey, claims.Role)
		c.Next()
	}
}

// OptionalBearerAuth attaches user id and role when a valid Bearer token is sent; otherwise continues as a guest (no context keys). If the client sends Authorization but the token is invalid, responds 401.
func OptionalBearerAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" {
			c.Next()
			return
		}
		const prefix = "Bearer "
		if !strings.HasPrefix(h, prefix) {
			api.WriteError(c, http.StatusUnauthorized, "unauthorized", "Authorization must be Bearer token")
			c.Abort()
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(h, prefix))
		if raw == "" {
			api.WriteError(c, http.StatusUnauthorized, "unauthorized", "empty bearer token")
			c.Abort()
			return
		}
		claims, err := auth.Parse(cfg, raw)
		if err != nil {
			api.WriteError(c, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			c.Abort()
			return
		}
		uid, err := auth.UserID(claims)
		if err != nil {
			api.WriteError(c, http.StatusUnauthorized, "unauthorized", "invalid token subject")
			c.Abort()
			return
		}
		c.Set(ContextUserIDKey, uid)
		c.Set(ContextRoleKey, claims.Role)
		c.Next()
	}
}

// RequireRole gates routes that must not rely on handler-local if-statements alone (defense in depth for admin deletes).
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, ok := c.Get(ContextRoleKey)
		if !ok {
			api.WriteError(c, http.StatusForbidden, "forbidden", "missing role")
			c.Abort()
			return
		}
		rs, _ := r.(string)
		if rs != role {
			api.WriteError(c, http.StatusForbidden, "forbidden", "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireAnyRole builds a set once per middleware closure so librarian/admin checks stay O(1) per request.
func RequireAnyRole(roles ...string) gin.HandlerFunc {
	allow := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allow[r] = struct{}{}
	}
	return func(c *gin.Context) {
		r, ok := c.Get(ContextRoleKey)
		if !ok {
			api.WriteError(c, http.StatusForbidden, "forbidden", "missing role")
			c.Abort()
			return
		}
		rs, _ := r.(string)
		if _, ok := allow[rs]; !ok {
			api.WriteError(c, http.StatusForbidden, "forbidden", "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}

// UserID reads typed identity from Gin context set by auth middleware (avoids repeating context key strings in handlers).
func UserID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(ContextUserIDKey)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

// Role returns the JWT role claim if BearerAuth or OptionalBearerAuth ran successfully.
func Role(c *gin.Context) (string, bool) {
	v, ok := c.Get(ContextRoleKey)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// MustBeSelfOrAdmin encodes “read my profile OR admin reads anyone” once for user handlers.
func MustBeSelfOrAdmin(c *gin.Context, target uuid.UUID) bool {
	role, ok := Role(c)
	if !ok {
		return false
	}
	if role == domain.RoleAdmin {
		return true
	}
	self, ok := UserID(c)
	if !ok {
		return false
	}
	return self == target
}
