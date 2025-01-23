package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

var DB *gorm.DB

func InitializeDB(db *gorm.DB) {
	DB = db
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type Receptionist struct {
	ID       int    `gorm:"primaryKey"`
	Name     string `gorm:"not null"`
	Email    string `gorm:"not null"`
	Username string `gorm:"unique: not null"`
	Password string `gorm:"not null"`
}

// Login handles user authentication
func Login(c *gin.Context) {
	var loginData LoginRequest
	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	var user Receptionist
	result := DB.Where("username = ?", loginData.Username).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if user.Password != loginData.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Set session cookie
	c.SetCookie(
		"session",
		user.Username,
		86400, // 24 hours
		"/",
		"",    // empty domain to work with both IP and domain name
		false, // set to true in production with HTTPS
		true,  // HttpOnly
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user": gin.H{
			"username": user.Username,
			"name":     user.Name,
		},
	})
}

// Logout handles user logout
func Logout(c *gin.Context) {
	// Clear the session cookie by setting MaxAge to -1
	c.SetCookie(
		"session",
		"",
		-1,
		"/",
		"",    // empty domain to work with both IP and domain name
		false, // set to true in production with HTTPS
		true,  // HttpOnly
	)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// CheckAuth middleware to verify authentication
func CheckAuth(c *gin.Context) {
	sessionCookie, err := c.Cookie("session")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		c.Abort()
		return
	}

	// Verify the user exists
	var user Receptionist
	result := DB.Where("username = ?", sessionCookie).First(&user)
	if result.Error != nil {
		c.SetCookie("session", "", -1, "/", "", false, true)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
		c.Abort()
		return
	}

	// Set user info in context for other handlers
	c.Set("username", user.Username)
	c.Set("user_id", user.ID)

	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"user": gin.H{
			"username": user.Username,
			"name":     user.Name,
		},
	})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionCookie, err := c.Cookie("session")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			c.Abort()
			return
		}

		// Verify the user exists
		var user Receptionist
		result := DB.Where("username = ?", sessionCookie).First(&user)
		if result.Error != nil {
			c.SetCookie("session", "", -1, "/", "", false, true)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
			c.Abort()
			return
		}

		// Set user info in context for other handlers
		c.Set("username", user.Username)
		c.Set("user_id", user.ID)
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
