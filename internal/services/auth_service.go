package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/caiofariavert/golang_vert_helper/internal/domain"
)

const (
	defaultAuthEmail       = "helper@vert-capital.com"
	defaultAuthPassword    = "Helper@123"
	defaultJWTSecret       = "helper-jwt-secret-change-me"
	defaultJWTTTLInMinutes = 60
)

// AuthClaims representa os claims do JWT usado nas rotas do helper.
type AuthClaims struct {
	Email       string `json:"email"`
	Name        string `json:"name"`
	IsSuperuser bool   `json:"is_superuser"`
	jwt.RegisteredClaims
}

// AuthService encapsula autenticacao de usuarios e emissao/validacao de JWT.
type AuthService struct {
	db       *gorm.DB
	jwtKey   []byte
	tokenTTL time.Duration
	logger   *slog.Logger
}

// NewAuthService cria um novo AuthService.
func NewAuthService(db *gorm.DB, logger *slog.Logger) *AuthService {
	if logger == nil {
		logger = slog.Default()
	}

	secret := strings.TrimSpace(os.Getenv("HELPER_JWT_SECRET"))
	if secret == "" {
		secret = defaultJWTSecret
	}

	ttlMinutes := defaultJWTTTLInMinutes
	if raw := strings.TrimSpace(os.Getenv("HELPER_JWT_TTL_MINUTES")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err == nil && parsed > 0 {
			ttlMinutes = parsed
		}
	}

	return &AuthService{
		db:       db,
		jwtKey:   []byte(secret),
		tokenTTL: time.Duration(ttlMinutes) * time.Minute,
		logger:   logger,
	}
}

// TokenTTL retorna o tempo de expiracao dos tokens emitidos.
func (s *AuthService) TokenTTL() time.Duration {
	return s.tokenTTL
}

// ProvisionDefaultUserFromEnv cria/atualiza automaticamente o usuario de autenticacao.
func (s *AuthService) ProvisionDefaultUserFromEnv(ctx context.Context) error {
	email := strings.TrimSpace(os.Getenv("HELPER_API_AUTH_EMAIL"))
	if email == "" {
		email = defaultAuthEmail
	}

	password := strings.TrimSpace(os.Getenv("HELPER_API_AUTH_PASSWORD"))
	if password == "" {
		password = defaultAuthPassword
	}

	return s.upsertAuthUser(ctx, email, password, "Helper", true)
}

// Authenticate valida credenciais e retorna um JWT Bearer.
func (s *AuthService) Authenticate(ctx context.Context, email, password string) (string, *domain.AuthUser, error) {
	normalizedEmail := strings.TrimSpace(strings.ToLower(email))
	if normalizedEmail == "" || strings.TrimSpace(password) == "" {
		return "", nil, domain.ErrInvalidCredentials
	}

	var user domain.AuthUser
	if err := s.db.WithContext(ctx).Where("email = ?", normalizedEmail).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, domain.ErrInvalidCredentials
		}
		return "", nil, err
	}

	if !user.Active {
		return "", nil, domain.ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, domain.ErrInvalidCredentials
	}

	now := time.Now()
	claims := &AuthClaims{
		Email:       user.Email,
		Name:        user.Name,
		IsSuperuser: user.IsSuperuser,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.jwtKey)
	if err != nil {
		return "", nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, &user, nil
}

// ParseToken valida um JWT e retorna os claims.
func (s *AuthService) ParseToken(tokenString string) (*AuthClaims, error) {
	claims := &AuthClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, domain.ErrInvalidToken
		}
		return s.jwtKey, nil
	})
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	if !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	return claims, nil
}

// GinMiddleware protege rotas exigindo Authorization: Bearer <token>.
func (s *AuthService) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "missing Authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid Authorization header format"})
			return
		}

		claims, err := s.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("helper_auth_user_id", claims.Subject)
		c.Set("helper_auth_email", claims.Email)
		c.Set("helper_auth_name", claims.Name)
		c.Set("helper_auth_is_superuser", claims.IsSuperuser)
		c.Next()
	}
}

func (s *AuthService) upsertAuthUser(ctx context.Context, email, password, name string, isSuperuser bool) error {
	normalizedEmail := strings.TrimSpace(strings.ToLower(email))
	if normalizedEmail == "" {
		return fmt.Errorf("auth user email is required")
	}
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("auth user password is required")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash auth user password: %w", err)
	}

	var existing domain.AuthUser
	err = s.db.WithContext(ctx).Where("email = ?", normalizedEmail).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		newUser := &domain.AuthUser{
			ID:           uuid.New().String(),
			Email:        normalizedEmail,
			PasswordHash: string(passwordHash),
			Name:         name,
			IsSuperuser:  isSuperuser,
			Active:       true,
		}
		if createErr := s.db.WithContext(ctx).Create(newUser).Error; createErr != nil {
			return createErr
		}

		s.logger.Info("default auth user created", "email", normalizedEmail)
		return nil
	}
	if err != nil {
		return err
	}

	existing.PasswordHash = string(passwordHash)
	existing.Name = name
	existing.IsSuperuser = isSuperuser
	existing.Active = true

	if err := s.db.WithContext(ctx).Save(&existing).Error; err != nil {
		return err
	}

	s.logger.Info("default auth user updated", "email", normalizedEmail)
	return nil
}
