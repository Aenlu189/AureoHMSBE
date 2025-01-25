package routes

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

type Rooms struct {
	ID     int    `gorm:"primaryKey"`
	Floor  int    `gorm:"not null"`
	Room   string `gorm:"unique: not null"`
	Status int    `gorm:"default: 1"`
}

type roomData struct {
	AvailableRooms int     `json:"availableRooms"`
	TotalRooms     int     `json:"totalRooms"`
	FullNight      int     `json:"fullNight"`
	DayCaution     int     `json:"dayCaution"`
	Session        int     `json:"session"`
	Housekeeping   int     `json:"housekeeping"`
	Maintenance    int     `json:"maintenance"`
	FoodRevenue    float64 `json:"foodRevenue"`
}

func GetDashboardStats(c *gin.Context) {
	var rooms []Rooms
	if err := DB.Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch room "})
		return
	}

	stats := roomData{
		TotalRooms: len(rooms),
	}

	for _, room := range rooms {
		switch room.Status {
		case 1:
			stats.AvailableRooms++
		case 2:
			stats.FullNight++
		case 3:
			stats.DayCaution++
		case 4:
			stats.Session++
		case 5:
			stats.Housekeeping++
		case 6:
			stats.Maintenance++
		}
	}

	// Get today's food revenue
	stats.FoodRevenue = GetDailyFoodRevenue()

	c.JSON(http.StatusOK, stats)
}

func GetRooms(c *gin.Context) {
	var room []Rooms
	if err := DB.Find(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, room)
}

func GetRoom(c *gin.Context) {
	roomNumber := c.Param("room")
	if roomNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Room number is required"})
		return
	}

	var room Rooms
	if err := DB.Where("room = ?", roomNumber).First(&room).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Room not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, room)
}

func UpdateRoomStatus(c *gin.Context) {
	roomNumber := c.Param("room")
	if roomNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Room number is required"})
		return
	}

	var room Rooms
	if err := c.BindJSON(&room); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	var existingRoom Rooms
	if err := DB.Where("room = ?", roomNumber).First(&existingRoom).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Room not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	if err := DB.Model(&existingRoom).Select("status").Updates(room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existingRoom)
}
