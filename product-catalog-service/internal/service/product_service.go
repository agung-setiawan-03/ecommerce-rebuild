package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
	"os"
	"product-catalog-service/internal/entity"
	"product-catalog-service/internal/repository"
)

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

type ProductService struct {
	productRepo repository.ProductRepository
	rdb         *redis.Client
}

func NewProductService(productRepo repository.ProductRepository, rdb *redis.Client) *ProductService {
	return &ProductService{
		productRepo: productRepo,
		rdb:         rdb,
	}
}

// GetProductStock retrieves the stock for product.
func (p *ProductService) GetProductStock(ctx context.Context, productID int) (int, error) {
	// Read data from cache
	key := fmt.Sprintf("product:%d", productID)
	productCache, err := p.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			logger.Warn().Msgf("Stock for product %d not found in cache", productID)
		} else {
			logger.Error().Msgf("Error getting product stock for product %d", productID)
			return 0, err
		}
	}

	if productCache != "" {
		var product entity.Product
		err := json.Unmarshal([]byte(productCache), &product)
		if err != nil {
			logger.Error().Msgf("Error unmarshalling product stock for product %d", productID)
			return 0, err
		}

		logger.Info().Msgf("Product stock for product %d: %d", productID, product.Stock)
		return product.Stock, nil
	}

	product, err := p.productRepo.GetProductById(productID)
	if err != nil {
		logger.Error().Err(err).Msgf("Error getting product by ID %d", productID)
		return 0, err
	}

	// Write data to cache
	err = p.rdb.Set(ctx, key, product, 0).Err()
	if err != nil {
		logger.Error().Err(err).Msgf("Error setting product stock for product %d", productID)
		return 0, err
	}
	return product.Stock, nil
}

// ReserveProductStock reserves stock for an order
func (p *ProductService) ReserveProductStock(ctx context.Context, productID int, quantity int) error {
	// Get product from cache
	key := fmt.Sprintf("product:%d", productID)
	productCache, err := p.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			logger.Warn().Msgf("Stock for product %d not found in cache", productID)
		} else {
			logger.Error().Msgf("Error getting product stock for product %d", productID)
			return err
		}
	}

	var productData entity.Product
	err = json.Unmarshal([]byte(productCache), &productData)
	if err != nil {
		logger.Error().Msgf("Error unmarshalling product stock for product %d", productID)
		return err
	}

	if productData.ID == 0 {
		product, err := p.productRepo.GetProductById(productID)
		if err != nil {
			logger.Error().Err(err).Msgf("Error getting product by ID %d", productData.ID)
			return err
		}
		productData = *product
	}

	if productData.Stock < quantity {
		logger.Warn().Msgf("Product %d out of stock", productID)
		return fmt.Errorf("product out of stock")
	}

	productData.Stock -= quantity
	_, err = p.productRepo.UpdateProduct(&productData)
	if err != nil {
		logger.Error().Err(err).Msgf("Error updating product %d", productID)
		return err
	}

	//Delete product from cache
	err = p.rdb.Del(ctx, key).Err()
	if err != nil {
		logger.Error().Err(err).Msgf("Error deleting product %d from cache", productID)
		return err
	}

	return nil
}

// ReleaseProductStock realeases reserved stock when an
func (p *ProductService) ReleaseProductStock(ctx context.Context, productID int, quantity int) error {
	key := fmt.Sprintf("product:%d", productID)
	productCache, err := p.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			logger.Warn().Msgf("Stock for product %d not found in cache", productID)
		} else {
			logger.Error().Msgf("Error getting product stock for product %d", productID)
			return err
		}
	}

	var productData entity.Product
	err = json.Unmarshal([]byte(productCache), &productData)
	if err != nil {
		logger.Error().Msgf("Error unmarshalling product stock for product %d", productID)
		return err
	}

	if productData.ID == 0 {
		product, err := p.productRepo.GetProductById(productID)
		if err != nil {
			logger.Error().Err(err).Msgf("Error getting product by ID %d", productID)
		}
		productData = *product
	}

	productData.Stock += quantity
	_, err = p.productRepo.UpdateProduct(&productData)
	if err != nil {
		logger.Error().Err(err).Msgf("Error updating product %d", productID)
		return err
	}

	// Delete product from cache
	err = p.rdb.Del(ctx, key).Err()
	if err != nil {
		logger.Error().Err(err).Msgf("Error deleting product %d from cache", productData.ID)
		return err
	}

	return nil
}
