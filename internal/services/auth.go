package services

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/helioschat/sync/internal/database"
	"github.com/helioschat/sync/internal/types"
	"golang.org/x/crypto/argon2"
)

const (
	// Argon2id parameters
	argon2Time    = 1
	argon2Memory  = 64 * 1024 // 64MB
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

type AuthService struct {
	jwtSecret []byte
	db        *database.RedisClient // Add Redis client for storing user data
}

func NewAuthService(jwtSecret string, db *database.RedisClient) *AuthService {
	return &AuthService{
		jwtSecret: []byte(jwtSecret),
		db:        db,
	}
}

// GenerateWallet creates a new wallet with a secure passphrase hash and salt
func (s *AuthService) GenerateWallet(passphrase string) (*types.Wallet, error) {
	if passphrase == "" {
		return nil, errors.New("passphrase cannot be empty")
	}

	uid := uuid.New()

	// Generate salt
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash passphrase with Argon2id
	hashedPassphrase := argon2.IDKey([]byte(passphrase), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	wallet := &types.Wallet{
		UID:              uid,
		Salt:             base64.StdEncoding.EncodeToString(salt),
		HashedPassphrase: base64.StdEncoding.EncodeToString(hashedPassphrase),
		CreatedAt:        time.Now(),
	}

	// Store wallet details (UID, salt, hashed passphrase) in Redis
	walletKey := fmt.Sprintf("wallet:%s", uid.String())
	walletData, err := types.WalletToJSON(wallet) // Assuming you have a helper to marshal
	if err != nil {
		return nil, fmt.Errorf("failed to marshal wallet: %w", err)
	}
	if err := s.db.Set(walletKey, string(walletData), 0); err != nil {
		return nil, fmt.Errorf("failed to save wallet: %w", err)
	}

	// Return only UID and CreatedAt to the client, not the salt or hash
	return &types.Wallet{UID: uid, CreatedAt: wallet.CreatedAt}, nil
}

// Login authenticates a user with their passphrase
func (s *AuthService) Login(userID uuid.UUID, passphrase string) (*types.AuthTokens, error) {
	if passphrase == "" {
		return nil, errors.New("passphrase is required")
	}

	// Retrieve wallet details from Redis
	walletKey := fmt.Sprintf("wallet:%s", userID.String())
	data, err := s.db.Get(walletKey)
	if err != nil {
		return nil, fmt.Errorf("user not found or failed to retrieve wallet: %w", err)
	}

	var storedWallet types.Wallet
	if err := types.WalletFromJSON([]byte(data), &storedWallet); err != nil { // Assuming you have a helper to unmarshal
		return nil, fmt.Errorf("failed to unmarshal wallet data: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(storedWallet.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	storedHashedPassphrase, err := base64.StdEncoding.DecodeString(storedWallet.HashedPassphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to decode stored hash: %w", err)
	}

	// Hash the provided passphrase with the stored salt
	currentHashedPassphrase := argon2.IDKey([]byte(passphrase), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Compare the hashes in constant time
	if subtle.ConstantTimeCompare(currentHashedPassphrase, storedHashedPassphrase) != 1 {
		return nil, errors.New("invalid passphrase")
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	tokens := &types.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour), // 24 hours
	}

	return tokens, nil
}

// ValidateToken validates a JWT token and returns the user ID
func (s *AuthService) ValidateToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return uuid.Nil, err
	}

	if !token.Valid {
		return uuid.Nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, errors.New("invalid token claims")
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return uuid.Nil, errors.New("user_id not found in token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user_id format: %w", err)
	}

	return userID, nil
}

// RefreshToken generates new tokens from a refresh token
func (s *AuthService) RefreshToken(refreshToken string) (*types.AuthTokens, error) {
	userID, err := s.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	accessToken, err := s.generateAccessToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.generateRefreshToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	tokens := &types.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	return tokens, nil
}

func (s *AuthService) generateAccessToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"type":    "access",
		"exp":     time.Now().Add(1 * time.Hour).Unix(), // 1 hour
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) generateRefreshToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"type":    "refresh",
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 days
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
