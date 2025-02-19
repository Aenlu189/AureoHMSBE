package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type WebsiteReservation struct {
	Name         string    `json:"name" binding:"required"`
	Phone        string    `json:"phone" binding:"required"`
	NationalID   string    `json:"nationalId"`
	CheckinDate  time.Time `json:"checkinDate" binding:"required"`
	CheckoutDate time.Time `json:"checkoutDate" binding:"required"`
	RoomType     string    `json:"roomType" binding:"required"`
	GuestCount   int       `json:"guestCount" binding:"required,min=1"`
	RoomCount    int       `json:"roomCount" binding:"required,min=1"`
	ExtraBed     bool      `json:"extraBed"`
	Notes        string    `json:"notes"`
}

func HandleWebsiteBooking(c *gin.Context) {
	var booking WebsiteReservation
	if err := c.ShouldBindJSON(&booking); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create a reservation record
	reservation := Reservation{
		Name:            booking.Name,
		Phone:           &booking.Phone,
		NationalID:      &booking.NationalID,
		RoomType:        booking.RoomType,
		GuestCount:      booking.GuestCount,
		RoomCount:       booking.RoomCount,
		CheckinDate:     booking.CheckinDate,
		CheckoutDate:    booking.CheckoutDate,
		ReservationDate: time.Now().UTC(),
		Status:          "CONFIRMED",
		ExtraBed:        booking.ExtraBed,
		PaymentType:     "NONE",
		Notes:           &booking.Notes,
	}

	if err := DB.Create(&reservation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create reservation",
			"details": err.Error(),
		})
		return
	}

	// Send confirmation email (implement later)
	// sendConfirmationEmail(booking)

	c.JSON(http.StatusOK, gin.H{
		"message":       "Reservation created successfully",
		"reservationId": reservation.ID,
	})
}
