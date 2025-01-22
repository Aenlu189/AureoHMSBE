package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type Staff struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Name      string    `gorm:"not null"`
	Email     string    `gorm:"not null"`
	Username  string    `gorm:"unique;not null"`
	Password  string    `gorm:"not null"`
	Role      string    `gorm:"type:enum('HOUSEKEEPING');default:'HOUSEKEEPING'"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

type CleaningRecord struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	RoomNumber string    `gorm:"not null"`
	StaffID    uint      `gorm:"not null"`
	Staff      Staff     `gorm:"foreignKey:StaffID"`
	StartTime  time.Time `gorm:"not null"`
	EndTime    *time.Time
	Status     string `gorm:"type:enum('IN_PROGRESS','COMPLETED');default:'IN_PROGRESS'"`
}

func StaffLogin(c *gin.Context) {
	var requestData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	var staff Staff
	result := DB.Where("username = ?", requestData.Username).First(&staff)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "Invalid username or password"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": result.Error.Error()})
		}
		return
	}

	if staff.Password != requestData.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid username or password"})
		return
	}

	token, err := GenerateToken(staff.ID, staff.Email, staff.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"staff": gin.H{
			"id":       staff.ID,
			"name":     staff.Name,
			"username": staff.Username,
			"role":     staff.Role,
		},
	})
}

func GetRoomsForCleaning(c *gin.Context) {
	var rooms []Rooms
	if err := DB.Where("status = ? OR status = ?", 5, 7).Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rooms"})
		return
	}

	var cleaningRecords []CleaningRecord
	if err := DB.Where("status = ?", "IN_PROGRESS").Find(&cleaningRecords).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cleaning records"})
		return
	}

	cleaningMap := make(map[string]CleaningRecord)
	for _, record := range cleaningRecords {
		cleaningMap[record.RoomNumber] = record
	}

	var response []gin.H
	for _, room := range rooms {
		roomData := gin.H{
			"number": room.Room,
			"floor":  room.Floor,
			"status": room.Status,
		}

		if record, exists := cleaningMap[room.Room]; exists {
			roomData["cleaning_start"] = record.StartTime
			roomData["staff_id"] = record.StaffID
		}

		response = append(response, roomData)
	}

	c.JSON(http.StatusOK, response)
}

func StartCleaning(c *gin.Context) {
	var request struct {
		RoomNumber string `json:"room_number"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	staffID := c.GetUint("staff_id") // From JWT middleware

	tx := DB.Begin()

	if err := tx.Model(&Rooms{}).Where("room = ?", request.RoomNumber).Update("status", 7).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update room status"})
		return
	}

	cleaningRecord := CleaningRecord{
		RoomNumber: request.RoomNumber,
		StaffID:    staffID,
		StartTime:  time.Now(),
		Status:     "IN_PROGRESS",
	}

	if err := tx.Create(&cleaningRecord).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cleaning record"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Cleaning started successfully"})
}

func CompleteCleaning(c *gin.Context) {
	var request struct {
		RoomNumber string `json:"room_number"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	staffID := c.GetUint("staff_id") // From JWT middleware

	tx := DB.Begin()

	if err := tx.Model(&Rooms{}).Where("room = ?", request.RoomNumber).Update("status", 1).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update room status"})
		return
	}

	now := time.Now()
	if err := tx.Model(&CleaningRecord{}).
		Where("room_number = ? AND staff_id = ? AND status = ?", request.RoomNumber, staffID, "IN_PROGRESS").
		Updates(map[string]interface{}{
			"end_time": now,
			"status":   "COMPLETED",
		}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update cleaning record"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Cleaning completed successfully"})
}

func GetStaffCleaningHistory(c *gin.Context) {
	staffID := c.GetUint("staff_id")

	var records []CleaningRecord
	if err := DB.Where("staff_id = ?", staffID).
		Order("created_at DESC").
		Limit(50).
		Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cleaning history"})
		return
	}

	c.JSON(http.StatusOK, records)
}
