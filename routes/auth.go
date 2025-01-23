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
	fmt.Printf("Login - Before setting session for user: %s\n", user.Username)
	fmt.Printf("Login - Request Origin: %s\n", c.GetHeader("Origin"))

	session.Clear()

	// Set session options based on the host
	host := c.Request.Host
	options := sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 24,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteNoneMode,
	}

	if host == "87.106.203.188:8080" {
		options.Domain = "87.106.203.188"
		fmt.Printf("Login - Setting production domain: %s\n", options.Domain)
	}

	session.Options(options)
	session.Set("user", user.Username)

	if err := session.Save(); err != nil {
		fmt.Printf("Failed to save session: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to save session"})
		return
	}

	// Verify session was saved
	savedUser := session.Get("user")
	fmt.Printf("Login - After saving session. Saved user: %v\n", savedUser)
	fmt.Printf("Login - Response Headers: %v\n", c.Writer.Header())

	c.JSON(http.StatusOK, gin.H{
		"message":  "Login successful!",
		"username": user.Username,
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
