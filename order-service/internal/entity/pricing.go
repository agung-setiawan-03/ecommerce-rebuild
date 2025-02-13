package entity

type Pricing struct {
	ProductID  int     `json:"product_id"`
	Markup     float64 `json:"markup"`      // Markup percentage
	Discount   float64 `json:"discount"`    // Discount percentage
	FinalPrice float64 `json:"final_price"` // Calculated final price
}
