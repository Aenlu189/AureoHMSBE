package routes

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type Staff struct {
	ID       int    `gorm:"primaryKey"`
	Name     string `gorm:"not null"`
	Email    string `gorm:"not null"`
	Username string `gorm:"unique; not null"`
	Password string `gorm:"not null"`
	Role     string `gorm:"type:enum('HOUSEKEEPING');default:'HOUSEKEEPING'"`
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

func generateStaffToken(staff Staff) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  staff.ID,
		"username": staff.Username,
		"name":     staff.Name,
		"role":     staff.Role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
	})

	return token.SignedString(jwtSecret)
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
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
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

	// Generate JWT token
	token, err := generateStaffToken(staff)
	if err != nil {
		fmt.Printf("Token generation error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful!",
		"token":   token,
		"staff": gin.H{
			"id":       staff.ID,
			"username": staff.Username,
			"name":     staff.Name,
			"role":     staff.Role,
		},
	})
}

func StaffAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization header required"})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set("user_id", claims["user_id"])
			c.Set("username", claims["username"])
			c.Set("name", claims["name"])
			c.Set("role", claims["role"])
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token claims"})
			c.Abort()
			return
		}
	}
}

func GetRoomsForCleaning(c *gin.Context) {
	staffId := c.GetFloat64("user_id")
	if staffId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

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
			roomData["cleaning_status"] = record.Status
			roomData["cleaning_start_time"] = record.StartTime
			if record.StaffID == uint(staffId) {
				roomData["assigned_to_me"] = true
			}
		}

		response = append(response, roomData)
	}

	c.JSON(http.StatusOK, gin.H{
		"rooms": response,
	})
}

func StartCleaning(c *gin.Context) {
	staffId := uint(c.GetFloat64("user_id"))
	if staffId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	var request struct {
		RoomNumber string `json:"room_number"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	tx := DB.Begin()

	var room Rooms
	if err := tx.Where("room = ? AND (status = ? OR status = ?)", request.RoomNumber, 5, 7).First(&room).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Room is not available for cleaning"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check room status"})
		}
		return
	}

	var existingRecord CleaningRecord
	if err := tx.Where("room_number = ? AND status = ?", request.RoomNumber, "IN_PROGRESS").First(&existingRecord).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Room is already being cleaned"})
		return
	} else if err != gorm.ErrRecordNotFound {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check cleaning status"})
		return
	}

	if err := tx.Model(&Rooms{}).Where("room = ?", request.RoomNumber).Update("status", 7).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update room status"})
		return
	}

	cleaningRecord := CleaningRecord{
		RoomNumber: request.RoomNumber,
		StaffID:    staffId,
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
	staffId := uint(c.GetFloat64("user_id"))
	if staffId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	var request struct {
		RoomNumber string `json:"room_number"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	tx := DB.Begin()

	var cleaningRecord CleaningRecord
	if err := tx.Where("room_number = ? AND staff_id = ? AND status = ?",
		request.RoomNumber, staffId, "IN_PROGRESS").First(&cleaningRecord).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No active cleaning record found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cleaning record"})
		}
		return
	}

	now := time.Now()
	cleaningRecord.EndTime = &now
	cleaningRecord.Status = "COMPLETED"

	if err := tx.Save(&cleaningRecord).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update cleaning record"})
		return
	}

	if err := tx.Model(&Rooms{}).Where("room = ?", request.RoomNumber).Update("status", 1).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update room status"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Cleaning completed successfully"})
}

func GetCleaningHistory(c *gin.Context) {
	staffId := uint(c.GetFloat64("user_id"))
	if staffId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	var records []CleaningRecord
	if err := DB.Where("staff_id = ?", staffId).
		Order("start_time DESC").
		Limit(50).
		Find(&records).Error; err != nil {
		fmt.Printf("Error fetching cleaning history: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cleaning history"})
		return
	}

	// Format the response with capitalized field names to match frontend expectations
	var formattedRecords []gin.H
	for _, record := range records {
		formattedRecord := gin.H{
			"RoomNumber": record.RoomNumber,
			"StartTime":  record.StartTime,
			"EndTime":    record.EndTime,
			"Status":     record.Status,
		}
		formattedRecords = append(formattedRecords, formattedRecord)
	}

	// Always return a records array, even if empty
	if formattedRecords == nil {
		formattedRecords = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"records": formattedRecords,
	})
}

// GetStaffList returns a list of all staff members
func GetStaffList(c *gin.Context) {
	var staffMembers []Staff
	if err := DB.Find(&staffMembers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staff list"})
		return
	}

	// Return only necessary information
	var staffList []gin.H
	for _, staff := range staffMembers {
		staffList = append(staffList, gin.H{
			"id":   staff.ID,
			"name": staff.Name,
		})
	}

	c.JSON(http.StatusOK, staffList)
}

// AssignStaffToRoom assigns a staff member to clean a specific room
func AssignStaffToRoom(c *gin.Context) {
	var request struct {
		RoomNumber string `json:"roomNumber"`
		StaffId    uint   `json:"staffId"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	tx := DB.Begin()

	// Check if room exists and is in housekeeping status
	var room Rooms
	if err := tx.Where("room = ? AND status = ?", request.RoomNumber, 5).First(&room).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Room not found or not available for cleaning"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check room status"})
		}
		return
	}

	// Check if staff exists
	var staff Staff
	if err := tx.First(&staff, request.StaffId).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Staff member not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check staff"})
		}
		return
	}

	// Check if room is already being cleaned
	var existingRecord CleaningRecord
	if err := tx.Where("room_number = ? AND status = ?", request.RoomNumber, "IN_PROGRESS").First(&existingRecord).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Room is already being cleaned"})
		return
	} else if err != gorm.ErrRecordNotFound {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check cleaning status"})
		return
	}

	// Create cleaning record
	cleaningRecord := CleaningRecord{
		RoomNumber: request.RoomNumber,
		StaffID:    request.StaffId,
		StartTime:  time.Now(),
		Status:     "IN_PROGRESS",
	}

	if err := tx.Create(&cleaningRecord).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cleaning record"})
		return
	}

	// Update room status to cleaning in progress
	if err := tx.Model(&Rooms{}).Where("room = ?", request.RoomNumber).Update("status", 7).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update room status"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{
		"message": "Staff assigned successfully",
		"staff": gin.H{
			"id":   staff.ID,
			"name": staff.Name,
		},
	})
}
