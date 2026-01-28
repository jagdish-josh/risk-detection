package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"risk-detection/internal/audit"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")
)

type service struct {
	repo      Repository
	jwtSecret string
	jwtTTL    time.Duration
	auditLog  *audit.Logger
}

func NewService(repo Repository, auditLog *audit.Logger, jwtSecret string, jwtTTL time.Duration) Service {
	return &service{
		repo:      repo,
		auditLog:  auditLog,
		jwtSecret: jwtSecret,
		jwtTTL:    jwtTTL,
	}
}

func (s *service) Signup(req SignupRequest, ipAddress string) (SignupResponse, error) {
	// Step 1: Validate password strength
	if len(req.Password) < 8 {
		return SignupResponse{}, ErrWeakPassword
	}

	// Step 2: Check if user already exists
	existingUser, err := s.repo.FindUserByEmail(req.Email)
	if err != nil {
		return SignupResponse{}, fmt.Errorf("find user: %w", err)
	}
	if existingUser != nil {
		return SignupResponse{}, ErrUserAlreadyExists
	}

	// Step 3: Hash password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return SignupResponse{}, fmt.Errorf("hash password: %w", err)
	}

	// Step 4: Create new user
	user := &User{
		Email:    req.Email,
		Password: string(hashedPassword),
		Role:     req.Role,
	}

	if err := s.repo.CreateUser(user); err != nil {
		return SignupResponse{}, fmt.Errorf("create user: %w", err)
	}

	// Step 5: Generate JWT token
	token, err := s.generateJWT(user)
	if err != nil {
		return SignupResponse{}, fmt.Errorf("generate token: %w", err)
	}

	// Step 6: Store device ID and IP address in user_security
	if err := s.repo.UpdateUserSecurity(user.ID, req.DeviceID, ipAddress); err != nil {
		return SignupResponse{}, fmt.Errorf("update security: %w", err)
	}
	s.auditLog.Log(audit.AuditLog{
		EventType:  audit.EventSecurityUpdated,
		Action:     "CREATE",
		EntityType: "user_security",
		ActorType:  "SYSTEM",
		NewValues: map[string]interface{}{
			"email": req.Email,
			"device_id": req.DeviceID,
		},
	})

	return SignupResponse{
		UserID:      user.ID,
		Email:       user.Email,
		Role:        user.Role,
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	}, nil
}

func (s *service) Login(req LoginRequest, ipAddress string) (LoginResponse, error) {
	// Step 1: Find user by email
	user, err := s.repo.FindUserByEmail(req.Email)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return LoginResponse{}, ErrInvalidCredentials
	}

	// Step 2: Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return LoginResponse{}, ErrInvalidCredentials
	}

	// Step 3: Generate JWT token
	token, err := s.generateJWT(user)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("generate token: %w", err)
	}

	// Step 4: Update device ID and IP address
	if err := s.repo.UpdateUserSecurity(user.ID, req.DeviceID, ipAddress); err != nil {
		return LoginResponse{}, fmt.Errorf("update security: %w", err)
	}
	s.auditLog.Log(audit.AuditLog{
		EventType:  audit.EventSecurityUpdated,
		Action:     "UPDATE",
		EntityType: "user_security",
		EntityID:   user.ID.String(),
		ActorType:  "SYSTEM",
		NewValues: map[string]interface{}{
			"device_id": req.DeviceID,
		},
	})
	s.auditLog.Log(audit.AuditLog{
		EventType:  audit.EventUserLogin,
		Action:     "LOGIN",
		EntityType: "users",
		EntityID:   user.ID.String(),
		Status:     "SUCCESS",
	})

	return LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	}, nil
}

func (s *service) generateJWT(user *User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID.String(),
		"role":  user.Role,
		"email": user.Email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(s.jwtTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
