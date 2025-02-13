package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"os"
	"time"
	"user-management-service/internal/entity"
	"user-management-service/internal/repository"
)

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

type UserService struct {
	repo repository.UserRepository
	rdb  *redis.Client
}

// NewUserService creates a new instance of UserService.
func NewUserService(repo repository.UserRepository) *UserService {
	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis server address
		Password: "",               // No password
		DB:       0,                // Default DB
	})

	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	
	return &UserService{repo: repo, rdb: rdb}
}

type JwtCustomClaims struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// GetUserByID retrieves a user by ID (stub for now).
func (s *UserService) GetUserByID(ctx context.Context, id int) (*entity.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		logger.Error().Err(err).Msgf("Error getting user by ID %d", id)
		return nil, err
	}

	return user, nil
}

// CreateUser creates a new user (stub for now).
func (s *UserService) CreateUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	createdUser, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		logger.Error().Err(err).Msg("Error creating user")
		return nil, err
	}

	return createdUser, nil
}

//// Login logs in a user with the given email and password.
//func (s *UserService) Login(email string, password string) (*entity.User, error) {
//	user, err := s.repo.GetUserByEmailAndPassword(email, password)
//	if err != nil {
//		logger.Error().Err(err).Msg("Error logging in user")
//		return nil, err
//	}
//
//	return user, nil
//}

func (s *UserService) Login(ctx context.Context, email, password string) (token string, err error) {
	user, err := s.repo.GetUserByEmailAndPassword(ctx, email, password)
	if err != nil {
		return "", err
	}

	// After validation, generate JWT token
	claims := &JwtCustomClaims{
		Name:  user.Username,
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}

	tkn := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := tkn.SignedString([]byte("secret"))
	if err != nil {
		return "", err
	}

	// Store the JWT token in Redis with the user email as the key
	err = s.rdb.Set(ctx, email, t, time.Hour*24).Err() // Set expiration to 24 hours
	if err != nil {
		return "", err
	}

	// Return the user and the generated JWT token
	return t, nil
}

func (s *UserService) ValidateToken(ctx context.Context, email string) (string, error) {
	// Retrieve the JWT token from Redis
	token, err := s.rdb.Get(ctx, email).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", fmt.Errorf("session not found")
		}
		return "", err
	}

	return token, nil
}
