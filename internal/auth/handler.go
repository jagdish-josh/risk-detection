package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Signup(ctx *gin.Context) {
	var req SignupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Signup(req, ctx.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrUserAlreadyExists):
			ctx.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		case errors.Is(err, ErrWeakPassword):
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "signup failed"})
		}
		return
	}

	ctx.JSON(http.StatusCreated, resp)
}

func (h *Handler) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Login(req, ctx.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}
