package routes

import (
	"errors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
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

	// Set simple session
	session := sessions.Default(c)
	session.Clear() // Clear any existing session first
	session.Set("user_id", user.ID)
	session.Set("authenticated", true)

	// Debug logging
	log.Printf("Setting session for user ID: %v", user.ID)
	log.Printf("Request headers: %v", c.Request.Header)

	if err := session.Save(); err != nil {
		log.Printf("Error saving session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error saving session"})
		return
	}

	log.Printf("Session saved successfully")
	log.Printf("Response headers: %v", c.Writer.Header())

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
	session.Save()
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func CheckAuth(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	authenticated := session.Get("authenticated")

	if userID == nil || authenticated == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Not authenticated"})
		return
	}

	var user Receptionist
	result := DB.First(&user, userID)
	if result.Error != nil {
		session.Clear()
		session.Save()
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
		userID := session.Get("user_id")
		authenticated := session.Get("authenticated")
		if userID == nil || authenticated == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Not authenticated"})
			c.Abort()
			return
		}

		// Add user info to context
		c.Set("user_id", userID)
		c.Set("username", session.Get("username"))
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

	session := sessions.Default(c)
	userID := session.Get("user_id")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Not authenticated"})
		return
	}

	var user Receptionist
	if err := DB.First(&user, userID).Error; err != nil {
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

func SetupSessionStore(router *gin.Engine, store sessions.Store) {
	router.Use(sessions.Sessions("mysession", store))
}
