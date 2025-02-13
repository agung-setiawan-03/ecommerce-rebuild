package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
	"time"
	"user-management-service/internal/api"
	"user-management-service/internal/repository"
	"user-management-service/internal/service"
)

func connectDB() (*sql.DB, error) {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/fc-db")
	if err != nil {
		return nil, err
	}
	return db, nil
}

func main() {
	db, err := connectDB()
	if err != nil {
		panic(err)
	}

	// Initialize UserService
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(*userRepo)
	userHandler := api.NewUserHandler(*userService)

	e := echo.New()

	config := middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(1),
				Burst:     3,
				ExpiresIn: 3 * time.Minute,
			}),
		IdentifierExtractor: func(context echo.Context) (string, error) {
			// for local
			return context.Request().RemoteAddr, nil
			// for production
			// return context.Request().Header.Get(echo.HeaderXRealIP), nil
			// return ctx.RealIP(), nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(429, map[string]string{"error": "rate limit exceeded"})
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(429, map[string]string{"error": "rate limit exceeded"})
		},
	}

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.RateLimiterWithConfig(config))

	// Routes
	e.GET("/users/:id", userHandler.GetUserByID)
	e.POST("/users", userHandler.CreateUser)
	e.POST("/login", userHandler.Login)
	e.GET("/users/validate", userHandler.ValidateSession)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
