package entity

type Order struct {
	ID              int              `json:"id"`
	UserID          int              `json:"user_id"`
	OrderID         int              `json:"order_id"`
	ProductRequests []ProductRequest `json:"product_requests"`
	Quantity        int              `json:"quantity"`
	Total           float64          `json:"total"`
	TotalMarkUp     float64          `json:"total_mark_up"`
	TotalDiscount   float64          `json:"total_discount"`
	Status          string           `json:"status"` // e.g., "created", "paid", "canceled"
	IdempotentKey   string           `json:"idempotent_key"`
}

type ProductRequest struct {
	ProductID  int     `json:"product_id"`
	Quantity   int     `json:"quantity"`
	MarkUp     float64 `json:"mark_up"`
	Discount   float64 `json:"discount"`
	FinalPrice float64 `json:"final_price"`
}

/*
Mysql Table

CREATE TABLE orders (
	id INT AUTO_INCREMENT PRIMARY KEY,
	user_id INT NOT NULL,


CREATE TABLE product_requests (
	id INT AUTO_INCREMENT PRIMARY KEY,
	order_id INT NOT NULL REFERENCES orders(id),
	product_id INT NOT NULL,
	quantity INT NOT NULL,
	mark_up DOUBLE NOT NULL,
	discount DOUBLE NOT NULL,
	final_price DOUBLE NOT NULL
);

*/
