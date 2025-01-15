package routes

import (
	"fmt"
	"github.com/gin-gonic/gin"
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
}

func CreateGuest(c *gin.Context) {
	var guest Guests

	if err := c.BindJSON(&guest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		fmt.Println(err.Error())
		return
	}

	if guest.CheckinDate.IsZero() {
		guest.CheckinDate = time.Now()
	}

	if err := DB.Create(&guest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create guest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Guest created successfully",
		"guest":   guest,
	})
}
