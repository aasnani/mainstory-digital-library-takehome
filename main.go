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
	"mainstory-digital-library-takehome/internal/handlers"
	"mainstory-digital-library-takehome/internal/middleware"
	"mainstory-digital-library-takehome/internal/repository"
	"mainstory-digital-library-takehome/internal/service"
)

func main() {
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

	userRepo := repository.NewUserRepository(pool)
	userSvc := service.NewUserService(cfg, userRepo)
	authH := handlers.NewAuthHandler(cfg, userSvc)
	userH := handlers.NewUsersHandler(userSvc)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CORS(cfg.CORSAllowOrigin))

	router.GET("/healthcheck", func(c *gin.Context) {
		c.String(http.StatusOK, "UP")
	})

	v1 := router.Group("/api/v1")
	v1.POST("/auth/register", authH.Register)
	v1.POST("/auth/login", authH.Login)

	authorized := v1.Group("")
	authorized.Use(middleware.BearerAuth(cfg))
	authorized.GET("/users/me", userH.Me)
	authorized.PATCH("/users/me", userH.PatchMe)
	authorized.GET("/users", userH.List)
	authorized.GET("/users/:id", userH.GetByID)
	authorized.PATCH("/users/:id", userH.PatchByID)
	authorized.DELETE("/users/:id", userH.DeleteByID)

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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
