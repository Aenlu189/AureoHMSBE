package routes

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type Income struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	Type       string    `gorm:"type:varchar(255);not null"` // Increased the length of the varchar to 255
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

	// Only validate that Type is not empty
	if income.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Type cannot be empty"})
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
		fmt.Printf("Error fetching today's income: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Failed to fetch income: %v", err)})
		return
	}
	c.JSON(http.StatusOK, incomes)
}

func GetIncomeByDate(c *gin.Context) {
	date := c.Param("date")
	var Incomes []Income

	if err := DB.Preload("Guest").Where("DATE(created_at) = ?", date).Order("created_at DESC").Find(&Incomes).Error; err != nil {
		fmt.Printf("Error fetching income by date: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Failed to fetch income: %v", err)})
		return
	}
	c.JSON(http.StatusOK, Incomes)
}
