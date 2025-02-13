package main

import (
	"database/sql"
	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql" // Import MySQL driver
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
	"product-catalog-service/internal/api"
	consumer2 "product-catalog-service/internal/consumer"
	"product-catalog-service/internal/repository"
	"product-catalog-service/internal/service"
	"time"
)

/*
REQUIREMENT DEPENDENCIES
1. go get github.com/labstack/echo/v4
2. go get github.com/golang-jwt/jwt/v5
3. go get github.com/labstack/echo/v4/middleware
4. go get github.com/labstack/echo-jwt/v4
5. go get github.com/go-redis/redis/v8
6. go get github.com/go-sql-driver/mysql
*/

// Database connection
func connectDB() (*sql.DB, error) {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/product_db")

	if err != nil {
		return nil, err
	}
	return db, nil
}

func main() {
	// Initialize database
	db, err := connectDB()
	if err != nil {
		panic(err)
	}

	//Initialize redis client
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Initialize product service
	productRepo := repository.NewProductRepository(db)
	productService := service.NewProductService(*productRepo, rdb)
	productHandler := api.NewProductHandler(*productService)

	// Consumer
	consumer := consumer2.NewConsumer(productService)
	go consumer.StartKafkaConsumer()

	// Initialize echo
	e := echo.New()

	// Rate Limiter
	config := middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(1),
				Burst:     3,
				ExpiresIn: 3 * time.Hour,
			}),
		IdentifierExtractor: func(context echo.Context) (string, error) {
			// for local
			return context.Request().RemoteAddr, nil

			// for production
			// return ctx.RealIP(), nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(429, map[string]string{"error": "Rate limit exceeded"})
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(429, map[string]string{"error": "Rate limit exceeded"})
		},
	}

	e.Use(middleware.Recover())
	e.Use(echojwt.JWT([]byte("secret")))
	e.Use(middleware.RateLimiterWithConfig(config))

	//Middleware
	e.Use(middleware.Logger())

	// Routes
	e.GET("/products/:id/stock", productHandler.GetProductStock)
	e.POST("/products/reserve", productHandler.ReserveProductStock)
	e.POST("/products/release", productHandler.ReleaseProductStock)

	// Start server
	e.Logger.Fatal(e.Start(":8081"))
}
