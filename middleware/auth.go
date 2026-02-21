package middleware

import (
	"net/http"
	"strings"
	"time"

	"food-delivery-api/config"
	"food-delivery-api/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint             `json:"user_id"`
	Email  string           `json:"email"`
	Role   models.UserRole  `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT for a given user
func GenerateToken(user *models.User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(config.JWTSecret)
}

// AuthRequired validates the JWT and injects claims into context
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required (Bearer <token>)"})
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return config.JWTSecret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", string(claims.Role))
		c.Next()
	}
}

// RoleRequired enforces that caller has one of the allowed roles
func RoleRequired(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role not found in context"})
			c.Abort()
			return
		}
		callerRole := models.UserRole(roleVal.(string))
		for _, r := range roles {
			if callerRole == r {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied. Required role(s): " + rolesString(roles),
		})
		c.Abort()
	}
}

func rolesString(roles []models.UserRole) string {
	s := ""
	for i, r := range roles {
		if i > 0 {
			s += ", "
		}
		s += string(r)
	}
	return s
}

// GetUserID extracts caller user ID from context
func GetUserID(c *gin.Context) uint {
	val, _ := c.Get("userID")
	return val.(uint)
}

// GetRole extracts caller role from context
func GetRole(c *gin.Context) models.UserRole {
	val, _ := c.Get("role")
	return models.UserRole(val.(string))
}
