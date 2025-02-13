package repository

import (
	"context"
	"database/sql"
	"dynamic-pricing/internal/entity"
	"errors"
	"fmt"
)

type PricingRepository struct {
	db *sql.DB
}

// NewRepository
func NewPricingRepository(db *sql.DB) *PricingRepository {
	return &PricingRepository{db: db}
}

// CreatePricingRule in database
func (r *PricingRepository) CreatePricingRule(ctx context.Context, rule *entity.PricingRule) error {
	query := `INSERT INTO pricing_rules(product_id, product_price, default_markup, default_discount, stock_threshold, markup_increase, discount_reduction)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, rule.ID, rule.ProductID, rule.ProductPrice, rule.DefaultMarkup, rule.DefaultDiscount, rule.StockThreshold, rule.MarkupIncrease, rule.DiscountReduction)
	return err
}

// UpdatePricingRule updates an existing pricing rule in the database
func (r *PricingRepository) UpdatePricingRule(ctx context.Context, rule *entity.PricingRule) error {
	query := `UPDATE pricing_rules SET product_price = ?, default_markup = ?, default_discount = ?, stock_threshold = ?, markup_increase = ?, discount_reduction = ? WHERE product_id = ?`
	_, err := r.db.ExecContext(ctx, query, rule.ProductPrice, rule.DefaultMarkup, rule.DefaultDiscount, rule.StockThreshold, rule.MarkupIncrease, rule.DiscountReduction, rule.ProductID)
	return err
}

// DeletePricingRule deletes a pricing rule from the database
func (r *PricingRepository) DeletePricingRule(ctx context.Context, productID int) error {
	query := `DELETE FROM pricing_rules WHERE product_id = ?`
	_, err := r.db.ExecContext(ctx, query, productID)
	return err
}

// GetPricingRule fetches the pricing rule for a specific product from the database
func (r *PricingRepository) GetPricingRule(ctx context.Context, productID int) (*entity.PricingRule, error) {
	query := `SELECT id, product_id, product_price, default_markup, default_discount, stock_threshold, markup_increase, discount_reduction 
		FROM pricing_rules WHERE product_id = ?`
	row := r.db.QueryRowContext(ctx, query, productID)
	var rule entity.PricingRule
	err := row.Scan(&rule.ID, &rule.ProductID, &rule.ProductPrice, &rule.DefaultMarkup, &rule.DefaultDiscount, &rule.StockThreshold, &rule.MarkupIncrease, &rule.DiscountReduction)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pricing rule not found for product %d", productID)
		}
		return nil, err
	}
	return &rule, nil
}
