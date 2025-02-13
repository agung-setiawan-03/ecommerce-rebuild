package service

import (
	"context"
	"dynamic-pricing/internal/entity"
	"dynamic-pricing/internal/repository"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"net/http"
)

// PricingService handles the pricing logic.
type PricingService struct {
	pricingRepo       *repository.PricingRepository
	productServiceURL string
	rdb               *redis.Client
}

// NewPricingService creates a new instance of PricingService.
func NewPricingService(pricingRepo *repository.PricingRepository, rdb *redis.Client, productServiceURL string) *PricingService {
	return &PricingService{
		pricingRepo:       pricingRepo,
		productServiceURL: productServiceURL,
		rdb:               rdb,
	}
}

// CalculatePricing calculates the final price for a product based on pricing rules.
func (s *PricingService) CalculatePricing(ctx context.Context, productID int) (*entity.Pricing, error) {
	// Step 1: Get the pricing rule for the product
	pricingRuleCacheKey := fmt.Sprintf("pricing_rule:%d", productID)
	pricingRuleCacheData, err := s.rdb.Get(ctx, pricingRuleCacheKey).Result()
	if err != nil {
		return nil, fmt.Errorf("could not fetch pricing rule from cache: %v", err)
	}

	var pricingRule *entity.PricingRule
	if pricingRuleCacheData != "" {
		if err := json.Unmarshal([]byte(pricingRuleCacheData), pricingRule); err != nil {
			return nil, fmt.Errorf("could not unmarshal pricing rule: %v", err)
		}
	} else {
		pricingRule, err = s.pricingRepo.GetPricingRule(ctx, productID)
		if err != nil {
			return nil, fmt.Errorf("could not fetch pricing rule: %v", err)
		}
	}

	if pricingRule == nil {
		return nil, fmt.Errorf("pricing rule not found for product %d", productID)
	}

	// Step 2: Check product stock
	available, err := s.checkProductStock(ctx, productID)
	if err != nil {
		return nil, err
	}

	// Step 3: Calculate price based on stock
	markup := pricingRule.DefaultMarkup
	discount := pricingRule.DefaultDiscount

	// If stock is below the threshold, apply price adjustments
	if available < pricingRule.StockThreshold {
		markup += pricingRule.MarkupIncrease
		discount -= pricingRule.DiscountReduction
	}

	// Step 4: Calculate the final price
	productPrice := pricingRule.ProductPrice
	finalPrice := productPrice * (1 + markup) * (1 - discount)

	// Step 5: Return the calculated pricing
	return &entity.Pricing{
		ProductID:  productID,
		Markup:     markup,
		Discount:   discount,
		FinalPrice: finalPrice,
	}, nil
}

// checkProductStock checks if the product is available in the required quantity.
func (s *PricingService) checkProductStock(ctx context.Context, productID int) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/products/%d/stock", s.productServiceURL, productID), nil)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("product not available")
	}

	var stockData map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&stockData); err != nil {
		return 0, err
	}

	availableStock := stockData["stock"]
	return availableStock, nil
}
