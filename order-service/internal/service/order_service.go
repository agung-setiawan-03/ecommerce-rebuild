package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
	"math/rand"
	"net/http"
	"order-service/internal/entity"
	"order-service/internal/repository"
	"os"
	"time"
)

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

// OrderService is a service that provides order-related operations
type OrderService struct {
	orderRepo         repository.OrderRepository
	productServiceURL string
	pricingServiceURL string
	kafkaWriter       *kafka.Writer
	rdb               *redis.Client
}

// NewOrderService creates a new instance of OrderService
func NewOrderService(orderRepo repository.OrderRepository, productServiceURL, pricingServiceURL string, kafkaWriter *kafka.Writer, rdb *redis.Client) *OrderService {
	return &OrderService{
		orderRepo:         orderRepo,
		productServiceURL: productServiceURL,
		pricingServiceURL: pricingServiceURL,
		kafkaWriter:       kafkaWriter,
		rdb:               rdb,
	}
}

/*
To implement concurrency using goroutines and channels for the CreateOrder function,
we need to perform the checkProductStock and getPricing for each product concurrently.
Each of these functions can be executed in its own goroutine, and we'll use channels to collect the results for each product.

Approach:
Asynchronous Execution:

We will use goroutines for both s.checkProductStock and s.getPricing for each product in the order.
This will allow the order creation to proceed without waiting for each product's availability check and pricing retrieval.

Channels for Communication:

We will use channels to collect the results of the availability check (checkProductStock) and pricing (getPricing).
Since both tasks are independent, they will be processed asynchronously using goroutines.

Error Handling:

We'll handle errors by using the channels to communicate back if any task fails.

*/

// CreateOrder creates a new order
func (s *OrderService) CreateOrder(ctx context.Context, order *entity.Order) (*entity.Order, error) {

	// get the idempotent key from order
	validate, err := s.validateIdempotentKey(ctx, order.IdempotentKey)
	if err != nil {
		return nil, err
	}

	if !validate {
		return nil, errors.New("idempotent key already exists")
	}

	order.OrderID = randomOrderID()

	availabilityCh := make(chan struct {
		ProductID int
		Available bool
		Error     error
	}, len(order.ProductRequests))

	pricingCh := make(chan struct {
		ProductID  int
		FinalPrice float64
		MarkUp     float64
		Discount   float64
		Error      error
	}, len(order.ProductRequests))

	for _, productRequest := range order.ProductRequests {
		//// check product availability
		//available, err := s.checkProductStock(productRequest.ProductID, productRequest.Quantity)
		//if err != nil {
		//	logger.Error().Err(err).Msgf("Error checking product stock for product %d", productRequest.ProductID)
		//	return nil, err
		//}
		//
		//// get pricing
		//pricing, err := s.getPricing(productRequest.ProductID)
		//if err != nil {
		//	logger.Error().Err(err).Msgf("Error getting pricing for product %d", productRequest.ProductID)
		//	return nil, err
		//}
		//
		//if !available {
		//	logger.Warn().Msgf("Product %d out of stock", productRequest.ProductID)
		//	return nil, fmt.Errorf("product out of stock")
		//}
		//
		//productRequest.FinalPrice = float64(productRequest.Quantity) * pricing.FinalPrice
		//productRequest.MarkUp = float64(productRequest.Quantity) * pricing.Markup
		//productRequest.Discount = float64(productRequest.Quantity) * pricing.Discount

		go func(productRequest *entity.ProductRequest) {
			available, err := s.checkProductStock(ctx, productRequest.ProductID, productRequest.Quantity)
			availabilityCh <- struct {
				ProductID int
				Available bool
				Error     error
			}{
				ProductID: productRequest.ProductID,
				Available: available,
				Error:     err,
			}
		}(&productRequest)

		go func(productRequest *entity.ProductRequest) {
			pricing, err := s.getPricing(ctx, productRequest.ProductID)
			pricingCh <- struct {
				ProductID  int
				FinalPrice float64
				MarkUp     float64
				Discount   float64
				Error      error
			}{
				ProductID:  productRequest.ProductID,
				FinalPrice: pricing.FinalPrice,
				MarkUp:     pricing.Markup,
				Discount:   pricing.Discount,
				Error:      err,
			}
		}(&productRequest)
	}

	for range order.ProductRequests {
		availabilityResult := <-availabilityCh
		pricingResult := <-pricingCh

		if availabilityResult.Error != nil {
			logger.Error().Err(availabilityResult.Error).Msgf("Error checking product stock for product %d", availabilityResult.ProductID)
			return nil, availabilityResult.Error
		}

		if !availabilityResult.Available {
			logger.Warn().Msgf("Product %d out of stock", availabilityResult.ProductID)
			return nil, fmt.Errorf("product out of stock")
		}

		if pricingResult.Error != nil {
			logger.Error().Err(pricingResult.Error).Msgf("Error getting pricing for product %d", pricingResult.ProductID)
			return nil, pricingResult.Error
		}

		for _, productRequest := range order.ProductRequests {
			if productRequest.ProductID == availabilityResult.ProductID {
				productRequest.FinalPrice = float64(productRequest.Quantity) * pricingResult.FinalPrice
				productRequest.MarkUp = float64(productRequest.Quantity) * pricingResult.MarkUp
				productRequest.Discount = float64(productRequest.Quantity) * pricingResult.Discount
			}
		}
	}

	order.Total = 0
	for _, productRequest := range order.ProductRequests {
		order.Total += productRequest.FinalPrice
	}

	createdOrder, err := s.orderRepo.CreateOrder(ctx, order)
	if err != nil {
		logger.Error().Err(err).Msg("Error creating order")
		return nil, err
	}

	// if env is set to test, return
	if os.Getenv("ENV") == "test" {
		return createdOrder, nil
	}
	err = s.publishOrderEvent(ctx, createdOrder, "created")
	if err != nil {
		return nil, err
	}

	return createdOrder, nil
}

// UpdateOrder updates an existing order
func (s *OrderService) UpdateOrder(ctx context.Context, order *entity.Order) (*entity.Order, error) {
	if order.Status == "paid" {
		// check product availability
		for _, productRequest := range order.ProductRequests {
			available, err := s.checkProductStock(ctx, productRequest.ProductID, productRequest.Quantity)
			if err != nil {
				logger.Error().Err(err).Msgf("Error checking product stock for product %d", productRequest.ProductID)
				return nil, err
			}

			if !available {
				logger.Warn().Msgf("Product %d out of stock", productRequest.ProductID)
				return nil, fmt.Errorf("product out of stock")
			}
		}
	}
	updateOrder, err := s.orderRepo.UpdateOrder(ctx, order)
	if err != nil {
		logger.Error().Err(err).Msg("Error updating order")
		return nil, err
	}

	err = s.publishOrderEvent(ctx, updateOrder, "updated")
	if err != nil {
		return nil, err
	}

	return updateOrder, nil
}

// CancelOrder cancels an existing order
func (s *OrderService) CancelOrder(ctx context.Context, id int) (*entity.Order, error) {
	order, err := s.orderRepo.GetOrderByID(ctx, id)
	if err != nil {
		logger.Error().Err(err).Msgf("Error getting order by ID %d", id)
		return nil, err
	}

	order.Status = "cancelled"

	updatedOrder, err := s.orderRepo.UpdateOrder(ctx, order)
	if err != nil {
		logger.Error().Err(err).Msg("Error updating order")
		return nil, err
	}

	err = s.publishOrderEvent(ctx, updatedOrder, "cancelled")
	if err != nil {
		return nil, err
	}

	return updatedOrder, nil
}

func (s *OrderService) checkProductStock(ctx context.Context, productId int, quantity int) (bool, error) {
	// if env is set to test, return true
	if os.Getenv("ENV") == "test" {
		return true, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/products/%d/stock", s.productServiceURL, productId), nil)
	if err != nil {
		return false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("product not available")
	}

	var stockData map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&stockData); err != nil {
		return false, err
	}

	availableStock := stockData["stock"]

	return availableStock >= quantity, nil
}

func (s *OrderService) getPricing(ctx context.Context, productId int) (*entity.Pricing, error) {
	// if env is set to test, return a default pricing
	if os.Getenv("ENV") == "test" {
		return &entity.Pricing{
			ProductID:  productId,
			Markup:     0.1,
			Discount:   0.05,
			FinalPrice: 100,
		}, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/products/%d/pricing", s.productServiceURL, productId), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get pricing")
	}

	var pricing entity.Pricing
	if err := json.NewDecoder(resp.Body).Decode(&pricing); err != nil {
		return nil, err
	}

	return &pricing, nil
}

func (s *OrderService) publishOrderEvent(ctx context.Context, order *entity.Order, key string) error {
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return err
	}

	// order-created-1 or order-updated-1
	msg := kafka.Message{
		Key:   []byte(fmt.Sprintf("order-%s-%d", key, order.ID)),
		Value: orderJSON,
	}

	err = s.kafkaWriter.WriteMessages(ctx, msg)
	if err != nil {
		return err
	}

	return nil
}

func (s *OrderService) validateIdempotentKey(ctx context.Context, key string) (bool, error) {
	// if env is set to test, return true
	if os.Getenv("ENV") == "test" {
		return true, nil
	}
	// check if the key exists in the redis cache
	// if it exists, return false
	redisKey := fmt.Sprintf("idempotent-key:%s", key)
	val, err := s.rdb.Get(ctx, redisKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return false, err
	}

	if val != "" {
		return false, errors.New("idempotent key already exists")
	}

	// if it doesn't exist, add the key to the cache with a TTL of 24 hours
	// and return true
	err = s.rdb.Set(ctx, redisKey, "exists", 24*time.Hour).Err()

	return true, nil
}

func randomOrderID() int {
	return 1000 + rand.Intn(1000)
}
