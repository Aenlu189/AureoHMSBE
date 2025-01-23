package routes

import (
	"errors"
	"fmt"
	"github.com/gin-contrib/sessions"
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

	session := sessions.Default(c)
	// Clear any existing session
	session.Clear()
	if err := session.Save(); err != nil {
		fmt.Printf("Error clearing session: %v\n", err)
	}

	fmt.Printf("Login - Before setting session for user: %s\n", user.Username)

	// Set session data
	session.Set("user", user.Username)
	session.Set("userID", user.ID)

	// Save immediately
	if err := session.Save(); err != nil {
		fmt.Printf("Error saving session: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error creating session"})
		return
	}

	fmt.Printf("Login - After setting session. Session data: %v\n", session.Get("user"))

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
	session := sessions.Default(c)
	session.Clear()
	fmt.Println("Session has been cleared")

	user := session.Get("user")
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to clear session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
	fmt.Println("Logged out successfully")
	fmt.Println("Session user: ", user)
}

func CheckSession(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("user")
	userID := session.Get("userID")

	fmt.Printf("CheckSession - Session data: user=%v, userID=%v\n", username, userID)

	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Not logged in"})
		return
	}

	// Get user details from database
	var user Receptionist
	result := DB.Where("username = ?", username).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Authenticated",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"name":     user.Name,
		},
	})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}
