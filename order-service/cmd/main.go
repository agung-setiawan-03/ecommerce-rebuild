package main

import (
	"database/sql"
	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql" // Import MySQL driver
	_ "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/segmentio/kafka-go"
	"golang.org/x/time/rate"
	"log"
	"order-service/internal/api"
	config2 "order-service/internal/config"
	"order-service/internal/repository"
	"order-service/internal/service"
	"order-service/internal/sharding"
	"order-service/migrations"
	"os"
	"time"
)

func connectDB() (*sql.DB, error) {
	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/order-db")
	if err != nil {
		return nil, err
	}
	return db, nil
}

func connectDBEnv(host, port, user, pass, dbname string) (*sql.DB, error) {
	dsn := user + ":" + pass + "@tcp(" + host + ":" + port + ")/" + dbname
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func main() {
	//db, err := connectDB()
	//if err != nil {
	//	panic(err)
	//}

	//db1, err := sql.Open("mysql", "root:@tcp(localhost:3306)/order-db-1")
	//if err != nil {
	//	panic(err)
	//}
	//
	//db2, err := sql.Open("mysql", "root:@tcp(localhost:3306)/order-db-2")
	//if err != nil {
	//	panic(err)
	//}
	//
	//db3, err := sql.Open("mysql", "root:@tcp(localhost:3306)/order-db-3")
	//if err != nil {
	//	panic(err)
	//}

	db1, err := connectDBEnv(os.Getenv("DB1_HOST"), os.Getenv("DB1_PORT"), os.Getenv("DB1_USER"), os.Getenv("DB1_PASS"), os.Getenv("DB1_NAME"))
	if err != nil {
		panic(err)
	}
	db2, err := connectDBEnv(os.Getenv("DB2_HOST"), os.Getenv("DB2_PORT"), os.Getenv("DB2_USER"), os.Getenv("DB2_PASS"), os.Getenv("DB2_NAME"))
	if err != nil {
		panic(err)
	}
	db3, err := connectDBEnv(os.Getenv("DB3_HOST"), os.Getenv("DB3_PORT"), os.Getenv("DB3_USER"), os.Getenv("DB3_PASS"), os.Getenv("DB3_NAME"))
	if err != nil {
		panic(err)
	}

	// Migrate tables
	err = migrations.AutoMigrateOrders(3, db1, db2, db3)
	if err != nil {
		log.Fatalf("Failed to migrate orders table: %v", err)
	}

	err = migrations.AutoMigrateProductRequests(3, db1, db2, db3)
	if err != nil {
		log.Fatalf("Failed to migrate product_requests table: %v", err)
	}

	//rdb := redis.NewClient(&redis.Options{
	//	Addr: "localhost:6379",
	//})

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	var kafkaWriter *kafka.Writer
	if os.Getenv("ENV") == "test" {
		kafkaWriter = nil
	} else {
		kafkaWriter = config2.NewKafkaWriter("order-topic")
	}

	router := sharding.NewShardRouter(3)

	orderRepo := repository.NewOrderRepository([]*sql.DB{db1, db2, db3}, router)
	orderService := service.NewOrderService(*orderRepo, "http://localhost:8081", "http://localhost:8083", kafkaWriter, rdb)
	orderHandler := api.NewOrderHandler(*orderService)

	e := echo.New()

	limiterConfig := middleware.RateLimiterConfig{
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
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	//e.Use(echojwt.JWT([]byte("secret"))) // uncomment this line to enable JWT middleware
	e.Use(middleware.RateLimiterWithConfig(limiterConfig))

	e.POST("/orders", orderHandler.CreateOrder)
	e.PUT("/orders", orderHandler.UpdateOrder)
	e.DELETE("/orders/:id", orderHandler.CancelOrder)

	e.Logger.Fatal(e.Start(":8082"))
}
