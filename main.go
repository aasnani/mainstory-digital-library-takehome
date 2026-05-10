// Command main wires HTTP routes to services: composition root only—business rules stay in internal/service.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"mainstory-digital-library-takehome/internal/config"
	"mainstory-digital-library-takehome/internal/db"
	"mainstory-digital-library-takehome/internal/domain"
	"mainstory-digital-library-takehome/internal/handlers"
	"mainstory-digital-library-takehome/internal/middleware"
	"mainstory-digital-library-takehome/internal/repository"
	"mainstory-digital-library-takehome/internal/service"
)

func main() {
	// Fail fast on missing secrets/DB URL so production never runs half-configured (JWT or Postgres would be undefined).
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	// Repositories take pgxpool so each request can borrow a connection without ad-hoc global state.
	userRepo := repository.NewUserRepository(pool)
	bookRepo := repository.NewBookRepository(pool)
	entRepo := repository.NewEntitlementRepository(pool)

	// Services depend on interfaces (stores) where tests swap fakes; handlers stay thin translators.
	userSvc := service.NewUserService(cfg, userRepo)
	bookSvc := service.NewBookService(bookRepo, entRepo)
	entSvc := service.NewEntitlementService(entRepo)

	authH := handlers.NewAuthHandler(cfg, userSvc)
	userH := handlers.NewUsersHandler(userSvc)
	bookH := handlers.NewBooksHandler(bookSvc)
	entH := handlers.NewEntitlementsHandler(entSvc)

	router := gin.New()
	router.Use(gin.Recovery())
	// CORS is explicit because SPAs (e.g. Lovable) call this API from another origin with Bearer headers.
	router.Use(middleware.CORS(cfg.CORSAllowOrigin))

	router.GET("/healthcheck", func(c *gin.Context) {
		c.String(http.StatusOK, "UP")
	})

	v1 := router.Group("/api/v1")
	v1.POST("/auth/register", authH.Register)
	v1.POST("/auth/login", authH.Login)

	// Catalog is its own group so browse works without login while still honoring JWT when the client sends one (marketing funnel + entitled reads).
	catalog := v1.Group("")
	catalog.Use(middleware.OptionalBearerAuth(cfg))
	catalog.GET("/books", bookH.List)
	catalog.GET("/books/:id", bookH.GetByID)

	// Everything else requires a valid token so user id/role come from JWT claims, never from request bodies.
	authorized := v1.Group("")
	authorized.Use(middleware.BearerAuth(cfg))
	authorized.GET("/users/me", userH.Me)
	authorized.GET("/users/me/library", bookH.MyLibrary)
	authorized.POST("/users/me/subscription/cancel", entH.CancelMySubscription)
	authorized.PATCH("/users/me", userH.PatchMe)
	authorized.GET("/users", userH.List)
	authorized.GET("/users/:id", userH.GetByID)
	authorized.PATCH("/users/:id", userH.PatchByID)
	authorized.DELETE("/users/:id", userH.DeleteByID)

	// Librarians curate; only admins delete so mistaken catalog wipes require elevated intent and match the API contract.
	libOrAdmin := []string{domain.RoleLibrarian, domain.RoleAdmin}
	authorized.POST("/books", middleware.RequireAnyRole(libOrAdmin...), bookH.Create)
	authorized.PATCH("/books/:id", middleware.RequireAnyRole(libOrAdmin...), bookH.Update)
	authorized.DELETE("/books/:id", middleware.RequireRole(domain.RoleAdmin), bookH.Delete)

	authorized.GET("/entitlements", entH.List)
	authorized.GET("/entitlements/:id", entH.GetByID)
	authorized.POST("/entitlements", entH.Create)
	authorized.PATCH("/entitlements/:id", middleware.RequireRole(domain.RoleAdmin), entH.Patch)

	// ReadHeaderTimeout bounds slowloris-style header reads without changing handler deadlines for normal JSON bodies.
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Graceful shutdown lets in-flight requests finish (within the timeout) instead of dropping connections on Ctrl+C or platform SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
