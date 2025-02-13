package entity

type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

/*
Schema MySQL for product table :
CREATE TABLE products (
	id int(11) NOT NULL AUTO_INCREMENT,
	name VARCHAR(255) NOT NULL,
	description text NOT NULL,
	price double NOT NULL,
	stock int(11) NOT NULL,
	PRIMARY KEY (id)
)ENGINE = InnoDB DEFAULT CHARSET=utf8mb4;
*/
