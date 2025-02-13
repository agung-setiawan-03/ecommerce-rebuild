package repository

import (
	"context"
	"database/sql"
	"user-management-service/internal/entity"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db}
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int) (*entity.User, error) {
	var user entity.User
	query := `SELECT id, username, email, password FROM users WHERE id = ?`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	query := `INSERT INTO users (username, email, password) VALUES (?, ?, ?)`
	res, err := r.db.ExecContext(ctx, query, user.Username, user.Email, user.Password)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	user.ID = int(id)
	return user, nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user *entity.User
	query := `SELECT id, username, email, password FROM users WHERE email = ?`
	err := r.db.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) GetUserByEmailAndPassword(ctx context.Context, email, password string) (*entity.User, error) {
	var user entity.User
	query := `SELECT id, username, email, password FROM users WHERE email = ? AND password = ?`
	err := r.db.QueryRowContext(ctx, query, email, password).Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
