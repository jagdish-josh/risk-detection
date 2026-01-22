package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {

		
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header missing",
			})
			return
		}

	
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header format must be Bearer <token>",
			})
			return
		}

		tokenStr := parts[1]
		claims := jwt.MapClaims{}

		token, err := jwt.ParseWithClaims(
			tokenStr,
			&claims, 
			func(token *jwt.Token) (interface{}, error) {
				if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			},
		)

		if err != nil {
			fmt.Printf("JWT Parse Error: %v\n", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid or expired token",
				"details": err.Error(),
			})
			return
		}

		if !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			return
		}

		
		userID, ok := claims["sub"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid sub claim",
			})
			return
		}

		role, _ := claims["role"].(string)
		email, _ := claims["email"].(string)

		//Store in Gin context
		c.Set("user_id", userID)
		c.Set("role", role)
		c.Set("email", email)

		
		c.Next()
	}
}
