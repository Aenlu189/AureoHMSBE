package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Rooms struct {
	ID     int    `gorm:"primaryKey"`
	Floor  int    `gorm:"not null"`
	Room   string `gorm:"unique: not null"`
	Status int    `gorm:"default: 1"`
}

type roomData struct {
	AvailableRooms int `json:"availableRooms"`
	TotalRooms     int `json:"totalRooms"`
	FullNight      int `json:"fullNight"`
	DayCaution     int `json:"dayCaution"`
	Session        int `json:"session"`
	Housekeeping   int `json:"housekeeping"`
	Maintenance    int `json:"maintenance"`
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
	c.JSON(http.StatusOK, stats)
}
