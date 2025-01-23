package routes

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

type Receptionist struct {
	ID       int    `gorm:"primaryKey"`
	Name     string `gorm:"not null"`
	Email    string `gorm:"not null"`
	Username string `gorm:"unique: not null"`
	Password string `gorm:"not null"`
}

var DB *gorm.DB

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
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid username or password"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		}
		return
	}

	if user.Password != requestData.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid username or password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"name":     user.Name,
		},
	})
}

func Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func CheckAuth(c *gin.Context) {
	// Frontend will handle auth check
	c.JSON(http.StatusOK, gin.H{"message": "Use client-side auth"})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Frontend will handle auth
		c.Next()
	}
}

func UpdatePassword(c *gin.Context) {
	var requestData struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := c.BindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	var user Receptionist
	if err := DB.First(&user, 1).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error finding user"})
		return
	}

	if user.Password != requestData.CurrentPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Current password is incorrect"})
		return
	}

	if err := DB.Model(&user).Update("password", requestData.NewPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error updating password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}
