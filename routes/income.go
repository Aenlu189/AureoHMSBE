package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type Income struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	Type       string    `gorm:"type:varchar(50);not null"` // Changed from enum to varchar to support custom types
	GuestID    uint      `gorm:"not null"`
	Guest      Guests    `gorm:"foreignKey:GuestID"`
	RoomNumber int       `gorm:"not null"`
	Amount     float64   `gorm:"not null"`
	CreatedAt  time.Time `gorm:"not null"`
}

func AddIncome(c *gin.Context) {
	var income Income
	if err := c.BindJSON(&income); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	// Validate Type field
	validTypes := map[string]bool{
		"CHECKED-IN":        true,
		"EXTEND-STAY":       true,
		"FOOD":              true,
		"EMPLOYEE_EATERIES": true,
		"GUEST_FOOD":        true,
		"ELECTRICITY":       true,
	}

	// If type is not in validTypes and not empty, it's considered a custom type
	if income.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Type cannot be empty"})
		return
	}

	// Check if it's a valid type
	if _, exists := validTypes[income.Type]; !exists {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid payment type"})
		return
	}

	income.CreatedAt = time.Now().UTC()
	if err := DB.Create(&income).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create income record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Income recorded successfully.",
		"income":  income,
	})
}

func GetTodayIncome(c *gin.Context) {
	var incomes []Income
	today := time.Now().UTC().Format("2006-01-02") // Format as YYYY-MM-DD

	if err := DB.Preload("Guest").Where("DATE(created_at) = ?", today).Order("created_at DESC").Find(&incomes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch income"})
		return
	}
	c.JSON(http.StatusOK, incomes)
}

func GetIncomeByDate(c *gin.Context) {
	date := c.Param("date")
	var Incomes []Income

	if err := DB.Preload("Guest").Where("DATE(created_at) = ?", date).Order("created_at DESC").Find(&Incomes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch income"})
		return
	}
	c.JSON(http.StatusOK, Incomes)
}
