package repository

import (
	"context"
	"database/sql"
	"order-service/internal/entity"
	"order-service/internal/sharding"
)

type OrderRepository struct {
	dbShards []*sql.DB
	router   *sharding.ShardRouter
}

func NewOrderRepository(dbShards []*sql.DB, router *sharding.ShardRouter) *OrderRepository {
	return &OrderRepository{dbShards, router}
}

func (r *OrderRepository) GetOrderByID(ctx context.Context, id int) (*entity.Order, error) {
	orderQuery := `SELECT id, user_id, quantity, total, status, total_mark_up, total_discount, order_id FROM orders WHERE id = ?`
	productRequestQuery := `SELECT product_id, quantity, mark_up, discount, final_price FROM product_requests WHERE order_id = ?`

	dbIndex := r.router.GetShard(id)
	db := r.dbShards[dbIndex]

	order := &entity.Order{}
	err := db.QueryRowContext(ctx, orderQuery, id).Scan(&order.ID, &order.UserID, &order.Quantity, &order.Total, &order.Status, &order.TotalMarkUp, &order.TotalDiscount, &order.OrderID)
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, productRequestQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		productRequest := entity.ProductRequest{}
		err := rows.Scan(&productRequest.ProductID, &productRequest.Quantity, &productRequest.MarkUp, &productRequest.Discount, &productRequest.FinalPrice)
		if err != nil {
			return nil, err
		}
		order.ProductRequests = append(order.ProductRequests, productRequest)
	}

	return order, nil
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order *entity.Order) (*entity.Order, error) {
	dbIndex := r.router.GetShard(order.OrderID)
	db := r.dbShards[dbIndex]

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Insert order
	orderQuery := `INSERT INTO orders (user_id, order_id, quantity, total, status, total_mark_up, total_discount, idempotent_key) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := tx.ExecContext(ctx, orderQuery, order.UserID, order.OrderID, order.Quantity, order.Total, order.Status, order.TotalMarkUp, order.TotalDiscount, order.IdempotentKey)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	orderID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	//// Insert product requests
	//productQuery := `
	//	INSERT INTO product_requests (order_id, product_id, quantity, mark_up, discount, final_price)
	//	VALUES (?, ?, ?, ?, ?, ?)`
	//for _, product := range order.ProductRequests {
	//	_, err := tx.Exec(productQuery, orderID, product.ProductID, product.Quantity, product.MarkUp, product.Discount, product.FinalPrice)
	//	if err != nil {
	//		tx.Rollback()
	//		return nil, err
	//	}
	//}

	// Insert product requests with batch
	productQuery := `
		INSERT INTO product_requests (order_id, product_id, quantity, mark_up, discount, final_price)
		VALUES `

	// Build the query
	var values []interface{}
	for _, product := range order.ProductRequests {
		productQuery += "(?, ?, ?, ?, ?, ?),"
		values = append(values, orderID, product.ProductID, product.Quantity, product.MarkUp, product.Discount, product.FinalPrice)
	}

	// Remove the trailing comma
	productQuery = productQuery[:len(productQuery)-1]

	// Execute the query batch insert
	_, err = tx.ExecContext(ctx, productQuery, values...)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	order.ID = int(orderID)
	return order, nil
}

func (r *OrderRepository) UpdateOrder(ctx context.Context, order *entity.Order) (*entity.Order, error) {
	dbIndex := r.router.GetShard(order.OrderID)
	db := r.dbShards[dbIndex]

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Update order
	orderQuery := `UPDATE orders SET user_id = ?, quantity = ?, total = ?, status = ?, total_mark_up = ?, total_discount = ? WHERE id = ?`
	_, err = tx.ExecContext(ctx, orderQuery, order.UserID, order.Quantity, order.Total, order.Status, order.TotalMarkUp, order.TotalDiscount, order.ID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Delete existing product requests
	deleteQuery := `DELETE FROM product_requests WHERE order_id = ?`
	_, err = tx.ExecContext(ctx, deleteQuery, order.ID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Insert product requests
	productQuery := `
		INSERT INTO product_requests (order_id, product_id, quantity, mark_up, discount, final_price)
		VALUES (?, ?, ?, ?, ?, ?)`
	for _, product := range order.ProductRequests {
		_, err := tx.ExecContext(ctx, productQuery, order.ID, product.ProductID, product.Quantity, product.MarkUp, product.Discount, product.FinalPrice)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (r *OrderRepository) DeleteOrder(ctx context.Context, id int) error {
	dbIndex := r.router.GetShard(id)
	db := r.dbShards[dbIndex]

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Delete product requests
	productQuery := `DELETE FROM product_requests WHERE order_id = ?`
	_, err = tx.ExecContext(ctx, productQuery, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Delete order
	orderQuery := `DELETE FROM orders WHERE id = ?`
	_, err = tx.ExecContext(ctx, orderQuery, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, id int, status string) error {
	dbIndex := r.router.GetShard(id)
	db := r.dbShards[dbIndex]

	query := `UPDATE orders SET status = ? WHERE id = ?`
	_, err := db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}

	return nil
}
