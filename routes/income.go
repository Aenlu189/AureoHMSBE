package routes

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type Income struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	Type       string    `gorm:"type:varchar(50);not null"`
	GuestID    uint      `gorm:"not null"`
	Guest      Guests    `gorm:"foreignKey:GuestID"`
	RoomNumber int       `gorm:"not null"`
	Amount     float64   `gorm:"not null"`
	CreatedAt  time.Time `gorm:"not null"`
}

func AddIncome(c *gin.Context) {
	var income Income
	if err := c.BindJSON(&income); err != nil {
		fmt.Printf("Error binding JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("Invalid request body: %v", err)})
		return
	}

	fmt.Printf("Received income request: %+v\n", income)

	// If type is not in validTypes and not empty, it's considered a custom type
	if income.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Type cannot be empty"})
		return
	}

	income.CreatedAt = time.Now().UTC()

	// Create a new record in the database
	result := DB.Create(&income)
	if result.Error != nil {
		fmt.Printf("Error creating income record: %v\n", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Failed to create income record: %v", result.Error)})
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
