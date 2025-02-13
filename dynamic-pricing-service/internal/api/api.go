package api

import (
	"dynamic-pricing/internal/service"
	"github.com/labstack/echo/v4"
)

// PricingHandler handles pricing-related requests.
type PricingHandler struct {
	pricingService *service.PricingService
}

// NewPricingHandler creates a new PricingHandler instance.
func NewPricingHandler(pricingService *service.PricingService) *PricingHandler {
	return &PricingHandler{
		pricingService: pricingService,
	}
}

// GetPricing handles the pricing request from the Order Service.
func (h *PricingHandler) GetPricing(c echo.Context) error {
	// Get the product ID and quantity from the request
	var pricingRequest struct {
		ProductID int `json:"product_id"`
	}

	if err := c.Bind(&pricingRequest); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request payload"})

	}

	// Calculate the pricing
	pricing, err := h.pricingService.CalculatePricing(c.Request().Context(), pricingRequest.ProductID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})

	}

	// Return the pricing
	return c.JSON(200, pricing)
}
