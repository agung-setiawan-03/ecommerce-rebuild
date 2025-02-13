package entity

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"` // In production, you'd store hashed passwords.
}

/*
Mysql Schema:
CREATE DATABASE user_management;
USE user_management;

CREATE TABLE users (
	id INT AUTO_INCREMENT PRIMARY KEY,
	username VARCHAR(50) NOT NULL,
	email VARCHAR(50) NOT NULL,
	password VARCHAR(255) NOT NULL
);

// create index for email and password
CREATE UNIQUE INDEX email_idx ON users(email);
CREATE INDEX password_idx ON users(password);



*/
