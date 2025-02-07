package routes

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type Receptionist struct {
	ID       int    `gorm:"primaryKey"`
	Name     string `gorm:"not null"`
	Email    string `gorm:"not null"`
	Username string `gorm:"unique: not null"`
	Password string `gorm:"not null"`
}

type Admin struct {
	ID       int    `gorm:"primaryKey"`
	Name     string `gorm:"not null"`
	Email    string `gorm:"not null"`
	Username string `gorm:"unique; not null"`
	Password string `gorm:"not null"`
}

var DB *gorm.DB
var jwtSecret = []byte("your-secret-key") // Change this to a secure secret key

func generateToken(user Receptionist) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"name":     user.Name,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
	})

	return token.SignedString(jwtSecret)
}

func generateAdminToken(admin Admin) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  admin.ID,
		"username": admin.Username,
		"name":     admin.Name,
		"isAdmin":  true,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
	})

	return token.SignedString(jwtSecret)
}

func Login(c *gin.Context) {
	var requestData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	var user Receptionist
	result := DB.Where("username = ?", requestData.Username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Invalid username or password"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": result.Error.Error()})
		}
		return
	}

	if user.Password != requestData.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid username or password"})
		return
	}

	// Generate JWT token
	token, err := generateToken(user)
	if err != nil {
		fmt.Printf("Token generation error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful!",
		"token":   token,
		"user": gin.H{
			"username": user.Username,
			"name":     user.Name,
		},
	})
}

func AdminLogin(c *gin.Context) {
	var requestData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	var admin Admin
	result := DB.Where("username = ?", requestData.Username).First(&admin)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Invalid admin credentials"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": result.Error.Error()})
		}
		return
	}

	if admin.Password != requestData.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid admin credentials"})
		return
	}

	// Generate JWT token
	token, err := generateAdminToken(admin)
	if err != nil {
		fmt.Printf("Admin token generation error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate admin token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Admin login successful!",
		"token":   token,
		"user": gin.H{
			"username": admin.Username,
			"name":     admin.Name,
			"isAdmin":  true,
		},
	})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization header required"})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set("user_id", claims["user_id"])
			c.Set("username", claims["username"])
			c.Set("name", claims["name"])
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token claims"})
			c.Abort()
			return
		}
	}
}

func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization header required"})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Check if the token is for an admin
			isAdmin, exists := claims["isAdmin"].(bool)
			if !exists || !isAdmin {
				c.JSON(http.StatusForbidden, gin.H{"message": "Admin access required"})
				c.Abort()
				return
			}

			// Store admin info in context
			c.Set("user_id", int(claims["user_id"].(float64)))
			c.Set("username", claims["username"].(string))
			c.Set("name", claims["name"].(string))
			c.Set("isAdmin", true)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token claims"})
			c.Abort()
			return
		}
	}
}

func Logout(c *gin.Context) {
	// With JWT, we don't need to do anything server-side for logout
	// The client should simply remove the token
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

func AdminCheckAuth(c *gin.Context) {
	// The AdminAuthMiddleware will handle the authentication check
	// If we reach here, it means the token is valid and the user is an admin
	c.JSON(http.StatusOK, gin.H{
		"message": "Admin authentication valid",
		"user": gin.H{
			"username": c.GetString("username"),
			"name":     c.GetString("name"),
			"isAdmin":  true,
		},
	})
}
