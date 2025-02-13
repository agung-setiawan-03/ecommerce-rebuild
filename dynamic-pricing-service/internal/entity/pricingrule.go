package entity

type PricingRule struct {
	ID                uint    `gorm:"primaryKey"`
	ProductID         int     `json:"product_id"`
	ProductPrice      float64 `json:"product_price"`
	DefaultMarkup     float64 `json:"default_markup"`
	DefaultDiscount   float64 `json:"default_discount"`
	StockThreshold    int     `json:"stock_threshold"`    // If stock is less than this, apply price adjustments
	MarkupIncrease    float64 `json:"markup_increase"`    // Increase markup by this percentage
	DiscountReduction float64 `json:"discount_reduction"` // Reduce discount by this percentage
}
