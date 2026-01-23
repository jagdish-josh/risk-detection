package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type service struct {
	repo      Repository
	jwtSecret string
	jwtTTL    time.Duration
}

func NewService(repo Repository, jwtSecret string, jwtTTL time.Duration) Service {
	return &service{
		repo:      repo,
		jwtSecret: jwtSecret,
		jwtTTL:    jwtTTL,
	}
}

func (s *service) Login(req LoginRequest, ipAdress string) (LoginResponse, error) {
	user, err := s.repo.FindUserByID(req.UserID)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return LoginResponse{}, ErrInvalidCredentials
	}

	// bytes, err := bcrypt.GenerateFromPassword(
	//     []byte(req.Password),
	//     bcrypt.DefaultCost,
	// )
	// fmt.Println(string(bytes))

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return LoginResponse{}, ErrInvalidCredentials
	}
	fmt.Println("authenticated")

	token, err := s.generateJWT(user)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("generate token: %w", err)
	}

	if err := s.repo.UpdateUserSecurity(user.ID, req.DeviceID, ipAdress); err != nil {
		return LoginResponse{}, fmt.Errorf("update security: %w", err)
	}

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
