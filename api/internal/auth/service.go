package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrValidationFailed   = errors.New("validation failed")
	ErrRateLimited        = errors.New("rate limited")
	ErrUnauthenticated    = errors.New("unauthenticated")
)

// Precomputed bcrypt cost 12 hash for timing attack mitigation when email is not found
const dummyBcryptHash = "$2a$12$7Eg1NnKzVq.9P7M8a5n6m.h/FzqK7Nn6WzC1gqR8z1hYk3O5gT7Ke"

type Service struct {
	repo        *Repo
	rateLimiter *RateLimiter
	sessionTTL  time.Duration
}

func NewService(repo *Repo, sessionTTLHours int) *Service {
	if sessionTTLHours <= 0 {
		sessionTTLHours = 720
	}
	return &Service{
		repo:        repo,
		rateLimiter: NewRateLimiter(5, time.Minute),
		sessionTTL:  time.Duration(sessionTTLHours) * time.Hour,
	}
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ValidateEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return false
	}
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func ValidatePassword(password string) bool {
	pwBytes := []byte(password)
	return len(pwBytes) >= 8 && len(pwBytes) <= 72
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

type AuthResult struct {
	User         *User
	SessionToken string
	ExpiresAt    time.Time
}

func (s *Service) Register(ctx context.Context, email, password string) (*AuthResult, error) {
	email = NormalizeEmail(email)
	if !ValidateEmail(email) {
		return nil, fmt.Errorf("%w: invalid email address", ErrValidationFailed)
	}
	if !ValidatePassword(password) {
		return nil, fmt.Errorf("%w: password must be between 8 and 72 bytes", ErrValidationFailed)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateUserWithDefaults(ctx, email, string(hash))
	if err != nil {
		return nil, err
	}

	token, err := GenerateSessionToken()
	if err != nil {
		return nil, err
	}
	tokenHash := HashToken(token)

	expiresAt := time.Now().UTC().Add(s.sessionTTL)
	if err := s.repo.CreateSession(ctx, tokenHash, user.ID, expiresAt.Format(time.RFC3339)); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &AuthResult{
		User:         user,
		SessionToken: token,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *Service) Login(ctx context.Context, ip, email, password string) (*AuthResult, error) {
	email = NormalizeEmail(email)
	rateKey := ip + ":" + email
	if !s.rateLimiter.Allow(rateKey) {
		return nil, ErrRateLimited
	}

	userWithHash, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Run dummy bcrypt to protect against timing attack
			_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userWithHash.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Login succeeded
	token, err := GenerateSessionToken()
	if err != nil {
		return nil, err
	}
	tokenHash := HashToken(token)

	expiresAt := time.Now().UTC().Add(s.sessionTTL)
	if err := s.repo.CreateSession(ctx, tokenHash, userWithHash.ID, expiresAt.Format(time.RFC3339)); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &AuthResult{
		User:         &userWithHash.User,
		SessionToken: token,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *Service) Logout(ctx context.Context, sessionToken string) error {
	if sessionToken == "" {
		return nil
	}
	tokenHash := HashToken(sessionToken)
	return s.repo.DeleteSession(ctx, tokenHash)
}

func (s *Service) GetCurrentUser(ctx context.Context, sessionToken string) (*User, error) {
	if sessionToken == "" {
		return nil, ErrUnauthenticated
	}
	tokenHash := HashToken(sessionToken)
	user, err := s.repo.GetSessionUser(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrUnauthenticated
		}
		return nil, err
	}
	return user, nil
}
