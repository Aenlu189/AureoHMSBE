package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type RoomPrices struct {
	ID         int     `json:"id" gorm:"primaryKey"`
	BNFP       float64 `json:"bnfp"`     // Full Night Price
	BCFP       float64 `json:"bcfp"`     // Caution Price
	BSFP       float64 `json:"bsfp"`     // Session Price
	ExtraBed   float64 `json:"ebed"`     // Extra Bed Price
	HourlyRate float64 `json:"eachHour"` // Hourly Rate
}

// GetRoomPrices retrieves the current room prices
func GetRoomPrices(c *gin.Context) {
	var prices RoomPrices
	result := DB.First(&prices)
	if result.Error != nil {
		// If no prices exist, return default prices
		prices = RoomPrices{
			BNFP:       63000,
			BCFP:       42000,
			BSFP:       30000,
			ExtraBed:   20000,
			HourlyRate: 10000,
		}
		// Create default prices in database
		DB.Create(&prices)
	}
	c.JSON(http.StatusOK, prices)
}

// UpdateRoomPrices updates the room prices
func UpdateRoomPrices(c *gin.Context) {
	var prices RoomPrices
	if err := c.ShouldBindJSON(&prices); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existingPrices RoomPrices
	result := DB.First(&existingPrices)
	if result.Error != nil {
		// If no prices exist, create new
		DB.Create(&prices)
	} else {
		// Update existing prices
		DB.Model(&existingPrices).Updates(prices)
	}

	c.JSON(http.StatusOK, prices)
}
