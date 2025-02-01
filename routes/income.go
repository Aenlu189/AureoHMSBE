package routes

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type Income struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	Type        string    `gorm:"column:type;type:varchar(255)"` // Changed to explicitly set column name and type
	GuestID     *uint     `gorm:"default:null"`                  // Changed to pointer to make it optional
	Guest       *Guests   `gorm:"foreignKey:GuestID"`            // Changed to pointer since it's optional
	RoomNumber  int       `gorm:"default:0"`                     // Made default 0
	Amount      float64   `gorm:"not null"`
	RevenueType string    `gorm:"column:revenue_type;type:varchar(50);default:'revenue'"` // Added revenue type field
	CreatedAt   time.Time `gorm:"not null"`
}

type IncomeRequest struct {
	Type        string  `json:"Type"`
	GuestID     uint    `json:"GuestID"`
	RoomNumber  int     `json:"RoomNumber"`
	Amount      float64 `json:"Amount"`
	RevenueType string  `json:"RevenueType"`
}

func AddIncome(c *gin.Context) {
	var req IncomeRequest
	if err := c.BindJSON(&req); err != nil {
		fmt.Printf("Error binding JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	fmt.Printf("Received income request: %+v\n", req)

	if req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Type cannot be empty"})
		return
	}

	// Create Income record with null GuestID if it's 0
	income := Income{
		Type:        req.Type,
		RoomNumber:  req.RoomNumber,
		Amount:      req.Amount,
		RevenueType: req.RevenueType,
		CreatedAt:   time.Now().UTC(),
	}

	// Only set GuestID if it's not 0
	if req.GuestID != 0 {
		guestID := req.GuestID
		income.GuestID = &guestID
	}

	if err := DB.Create(&income).Error; err != nil {
		fmt.Printf("Error creating income: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Failed to create income record: %v", err)})
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
