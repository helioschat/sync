package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid" // Added for UUID parsing
	"github.com/helioschat/sync/internal/services"
	"github.com/helioschat/sync/internal/types"
)

type AuthHandler struct {
	AuthService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		AuthService: authService,
	}
}

// GenerateWallet creates a new wallet with passphrase
func (h *AuthHandler) GenerateWallet(c *gin.Context) {
	var req struct {
		Passphrase string `json:"passphrase" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid request format: passphrase is required",
				Details: err.Error(),
			},
		})
		return
	}

	wallet, err := h.AuthService.GenerateWallet(req.Passphrase)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusInternalServerError,
				Message: "Failed to generate wallet",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data: gin.H{
			"uid":        wallet.UID.String(), // Ensure UID is stringified
			"created_at": wallet.CreatedAt.Format(time.RFC3339Nano),
		},
	})
}

// Login authenticates a user with their passphrase
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		UserID     string `json:"user_id" binding:"required"`
		Passphrase string `json:"passphrase" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid request format: user_id and passphrase are required",
				Details: err.Error(),
			},
		})
		return
	}

	parsedUID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid user_id format",
				Details: err.Error(),
			},
		})
		return
	}

	tokens, err := h.AuthService.Login(parsedUID, req.Passphrase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusUnauthorized,
				Message: "Authentication failed",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data: gin.H{
			"tokens":  tokens,
			"user_id": parsedUID.String(), // Return the parsed and stringified UID
		},
	})
}

// RefreshToken generates new tokens from a refresh token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	tokens, err := h.AuthService.RefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusUnauthorized,
				Message: "Invalid refresh token",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    tokens,
	})
}
