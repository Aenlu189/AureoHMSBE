package routes

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type Guests struct {
	ID           int       `gorm:"primaryKey;autoIncrement"`
	Name         string    `gorm:"not null"`
	NationalID   *string   `gorm:"null"`
	Phone        *string   `gorm:"null"`
	RoomType     string    `gorm:"type:enum('FULL-NIGHT','DAY-CAUTION','SESSION');default:'FULL-NIGHT';not null"`
	RoomNumber   int       `gorm:"not null"`
	CheckinDate  time.Time `gorm:"type:datetime;not null"`
	CheckoutDate time.Time `gorm:"type:datetime;not null"`
	ExtraBed     bool      `gorm:"default:false"`
	PaymentType  string    `gorm:"type:enum('NONE', 'KPAY', 'AYAPAY', 'WAVEPAY', 'CASH');default:'NONE'"`
	AmountPaid   *int      `gorm:"null"`
	ExtraCharges int       `gorm:"not null; default:0"`
	FoodCharges  int       `gorm:"not null; default:0"`
	Paid         bool      `gorm:"default:false"`
	Status       string    `gorm:"type:enum('ACTIVE', 'CHECKED-OUT'); default:'ACTIVE'"`
}

func CreateGuest(c *gin.Context) {
	var guest Guests

	if err := c.BindJSON(&guest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		fmt.Println(err.Error())
		return
	}

	// No need to set CheckinDate here since it's already set in frontend
	// with exact Myanmar time down to seconds

	if err := DB.Create(&guest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create guest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Guest created successfully",
		"guest":   guest,
	})
}

func GetCurrentGuest(c *gin.Context) {
	roomNumber := c.Param("roomNumber")

	var guest Guests
	var allGuests []Guests
	DB.Where("room_number = ?", roomNumber).Find(&allGuests)
	fmt.Printf("Found %d guests for room %s\n", len(allGuests), roomNumber)
	for _, g := range allGuests {
		fmt.Printf("Guest: %+v\n", g)
	}

	if err := DB.Where("room_number = ? AND status = ?",
		roomNumber, "ACTIVE").First(&guest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "No active guest found in this room"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch guest: " + err.Error()})
		return
	}
	fmt.Println("Active guest found:", guest)
	c.JSON(http.StatusOK, guest)
}

func UpdateGuestInfo(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Guest ID is required."})
		return
	}

	var guest Guests
	if err := c.BindJSON(&guest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	var existingGuest Guests
	if err := DB.First(&existingGuest, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Guest not found."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	if err := DB.Model(&existingGuest).Updates(guest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existingGuest)
}

func UpdateGuestFoodPrice(c *gin.Context) {
	guestID := c.Param("id")
	fmt.Printf("Received guestID: %s\n", guestID)

	if guestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Guest ID is required"})
		return
	}

	var guest Guests
	if err := DB.First(&guest, guestID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Guest not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch guest"})
		return
	}

	var requestBody struct {
		FoodCharges float64 `json:"foodCharges"`
		AmountPaid  float64 `json:"amountPaid"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	// Update both food charges and amount paid
	updates := map[string]interface{}{
		"food_charges": requestBody.FoodCharges,
		"amount_paid":  requestBody.AmountPaid,
	}

	if err := DB.Model(&guest).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update guest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Guest updated successfully",
		"guest":   guest,
	})
}

func GetTodayCheckouts(c *gin.Context) {
	var guests []Guests
	now := time.Now().UTC()

	// Create start and end of UTC day
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, time.UTC)

	if err := DB.Where("checkout_date BETWEEN ? AND ? AND Status = ?", startOfDay, endOfDay, "ACTIVE").Find(&guests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch checkouts"})
		return
	}

	c.JSON(http.StatusOK, guests)
}
