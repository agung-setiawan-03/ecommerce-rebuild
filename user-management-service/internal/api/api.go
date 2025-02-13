package api

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"strconv"
	"user-management-service/internal/entity"
	"user-management-service/internal/service"
)

type UserHandler struct {
	userService service.UserService
}

// NewUserHandler creates a new instance of UserHandler
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type JwtCustomClaims struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// GetUserByID retrieves a user by ID --> /users/:id
func (h *UserHandler) GetUserByID(c echo.Context) error {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid ID"})
	}
	user, err := h.userService.GetUserByID(c.Request().Context(), idInt)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, user)
}

// CreateUser creates a new user --> /users
func (h *UserHandler) CreateUser(c echo.Context) error {
	user := entity.User{}
	if err := c.Bind(&user); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request payload"})
	}

	createdUser, err := h.userService.CreateUser(c.Request().Context(), &user)

	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, createdUser)
}

// Login logs in a user --> /users/login
func (h *UserHandler) Login(c echo.Context) error {
	ctx := c.Request().Context()

	login := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}

	if err := c.Bind(&login); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request payload"})
	}

	token, err := h.userService.Login(ctx, login.Email, login.Password)
	if err != nil {
		return c.JSON(401, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"token": token})
}

// ValidateSession validates a session token --> /users/validate
func (h *UserHandler) ValidateSession(c echo.Context) error {
	ctx := c.Request().Context()

	token := c.Request().Header.Get("Authorization")
	if token == "" {
		return c.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	validateToken, err := h.userService.ValidateToken(ctx, token)
	if err != nil {
		return c.JSON(401, map[string]string{"error": err.Error()})
	}

	if validateToken != token {
		return c.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	return c.JSON(200, map[string]string{"message": "Session is valid"})
}
