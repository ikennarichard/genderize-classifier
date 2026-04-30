package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/ikennarichard/genderize-classifier/internal/domain"
	"github.com/ikennarichard/genderize-classifier/internal/repository"
)

type TokenService struct {
	jwtSecret   []byte
	userRepo    domain.UserRepository
	sessionRepo repository.SessionRepository
}

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func NewTokenService(secret string, userRepo domain.UserRepository, sessionRepo repository.SessionRepository) *TokenService {
	return &TokenService{
		jwtSecret:   []byte(secret),
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

func (s *TokenService) generateToken(user *domain.User, duration time.Duration) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *TokenService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

func (s *TokenService) GenerateTokenPair(ctx context.Context, user *domain.User) (string, string, error) {
	accessString, err := s.generateToken(user, 3*time.Minute)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshString := uuid.New().String()
	expiry := time.Now().Add(5 * time.Minute)

    if err := s.sessionRepo.CreateSession(ctx, user.ID, refreshString, expiry); err != nil {
        return "", "", fmt.Errorf("failed to create session: %w", err)
    }

	return accessString, refreshString, nil
}

func (s *TokenService) RotateRefreshToken(ctx context.Context, oldToken string) (string, string, error) {
	 session, err := s.sessionRepo.FindSession(ctx, oldToken)
	if err != nil {
		return "", "", fmt.Errorf("session not found: %w", err)
	}

	if session.IsRevoked || time.Now().After(session.ExpiresAt) {
		 _ = s.sessionRepo.RevokeAllUserSessions(ctx, session.UserID)
		return "", "", fmt.Errorf("invalid or expired session")
	}

	user, err := s.userRepo.FindByID(session.UserID)
	if err != nil || !user.IsActive {
		return "", "", fmt.Errorf("user unauthorized")
	}

	if err := s.sessionRepo.RevokeSession(ctx, oldToken); err != nil {
		return "", "", fmt.Errorf("failed to revoke old session: %w", err)
	}

	return s.GenerateTokenPair(ctx, user)
}

func (s *TokenService) RevokeSession(ctx context.Context, token string) error {
	if token == "" {
		return errors.New("token is required")
	}
	 return s.sessionRepo.RevokeSession(ctx, token)
}