package main

import (
	"database/sql"
	"dynamic-pricing/internal/api"
	"dynamic-pricing/internal/repository"
	"dynamic-pricing/internal/service"
	"github.com/go-redis/redis/v8"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func connectDB() (*sql.DB, error) {
	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/dynamic-pricing-db")
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

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Initialize product service
	pricingRepo := repository.NewPricingRepository(db)
	pricingService := service.NewPricingService(pricingRepo, rdb, "http://localhost:8081")
	pricingHandler := api.NewPricingHandler(pricingService)

	// Initialize echo
	e := echo.New()
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(echojwt.JWT([]byte("secret")))

	// Routes
	e.POST("/pricing", pricingHandler.GetPricing)
}
