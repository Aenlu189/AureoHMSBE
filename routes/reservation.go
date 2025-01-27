package routes

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type Reservation struct {
	ID              int       `gorm:"primaryKey;autoIncrement"`
	Name            string    `gorm:"not null"`
	NationalID      *string   `gorm:"null"`
	Phone           *string   `gorm:"null"`
	RoomType        string    `gorm:"type:enum('FULL-NIGHT','DAY-CAUTION','SESSION');default:'FULL-NIGHT';not null"`
	GuestCount      int       `gorm:"not null"`
	RoomCount       int       `gorm:"not null"`
	CheckinDate     time.Time `gorm:"type:date;not null"`
	CheckoutDate    time.Time `gorm:"type:date;not null"`
	ReservationDate time.Time `gorm:"type:date;not null"`
	Status          string    `gorm:"type:enum('CANCELLED','CHECKED-IN','CONFIRMED');default:'CONFIRMED'"`
	ExtraBed        bool      `gorm:"default:false"`
	PaymentType     string    `gorm:"type:enum('NONE', 'KPAY', 'AYAPAY', 'WAVEPAY', 'CASH');default:'NONE'"`
	AmountPaid      *int      `gorm:"null"`
	Notes           *string   `gorm:"type:text;null"`
}

func CreateReservation(c *gin.Context) {
	var reservation Reservation

	if err := c.BindJSON(&reservation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	if reservation.ReservationDate.IsZero() {
		reservation.ReservationDate = time.Now().UTC()
	}

	if err := DB.Create(&reservation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create reservation: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":     "Reservation created successfully",
		"reservation": reservation,
	})
}

func GetReservationsByDate(c *gin.Context) {
	date := c.Param("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Check-in date is required"})
		return
	}

	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid date format"})
		return
	}

	var reservations []Reservation

	// Use UTC time for date comparison
	query := DB.Where("DATE(reservation_date) = ?", parsedDate)
	if err := query.Find(&reservations).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "No reservations found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch reservations"})
		return
	}

	c.JSON(http.StatusOK, reservations)
}

func GetReservation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Reservation ID is required"})
		return
	}

	var reservation Reservation
	if err := DB.First(&reservation, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Reservation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
	}
	c.JSON(http.StatusOK, reservation)
}

func DeleteReservation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ID of guest is required"})
		return
	}

	var reservation []Reservation
	if err := DB.First(&reservation, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Reservation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	if err := DB.Delete(&reservation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete reservation"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Reservation deleted successfully"})
}

func UpdateReservation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Reservation ID is required"})
		return
	}

	var reservation Reservation
	if err := c.BindJSON(&reservation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	var existingReservation Reservation
	if err := DB.First(&existingReservation, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Reservation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	if err := DB.Model(&existingReservation).Updates(reservation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existingReservation)
}
