package migrations

import (
	"database/sql"
	"time"
)

// AutoMigrateOrders creates the orders table if it does not exist.
func AutoMigrateOrders(retries int, dbs ...*sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS orders (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT NOT NULL,
			order_id INT NOT NULL UNIQUE,
			quantity INT NOT NULL,
			total DOUBLE NOT NULL,
			total_mark_up DOUBLE NOT NULL,
			total_discount DOUBLE NOT NULL,
			status VARCHAR(20) NOT NULL,
			idempotent_key VARCHAR(255) UNIQUE NOT NULL
		);
	`
	for _, db := range dbs {
		_, err := db.Exec(query)
		if err != nil {
			// Retry creating the table
			for i := 0; i < retries; i++ {
				time.Sleep(1 * time.Second)
				_, err = db.Exec(query)
				if err == nil {
					break
				}
			}
		}
	}
	return nil
}

// AutoMigrateProductRequests creates the product_requests table if it does not exist.
func AutoMigrateProductRequests(retries int, dbs ...*sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS product_requests (
			id INT AUTO_INCREMENT PRIMARY KEY,
			order_id INT NOT NULL,
			product_id INT NOT NULL,
			quantity INT NOT NULL,
			mark_up DOUBLE NOT NULL,
			discount DOUBLE NOT NULL,
			final_price DOUBLE NOT NULL,
			FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
		);
	`
	for _, db := range dbs {
		_, err := db.Exec(query)
		if err != nil {
			// Retry creating the table
			for i := 0; i < retries; i++ {
				time.Sleep(1 * time.Second)
				_, err = db.Exec(query)
				if err == nil {
					break
				}
			}
		}
	}
	return nil
}
