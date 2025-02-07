package routes

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"sort"
	"time"
)

type RevenueData struct {
	TotalRevenue float64   `json:"totalRevenue"`
	RoomRevenue  float64   `json:"roomRevenue"`
	FoodRevenue  float64   `json:"foodRevenue"`
	OtherRevenue float64   `json:"otherRevenue"`
	Date         time.Time `json:"date"`
}

type Activity struct {
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Amount      float64   `json:"amount"`
	RoomNumber  int       `json:"roomNumber,omitempty"`
	GuestID     uint      `json:"guestId,omitempty"`
	Description string    `json:"description,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

func GetFoodOrdersByDate(c *gin.Context) {
	date := c.Param("date")
	var foodOrders []FoodOrder

	if err := DB.Where("DATE(created_at) = ?", date).
		Order("created_at DESC").
		Find(&foodOrders).Error; err != nil {
		fmt.Printf("Error fetching food orders: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch food orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"foodOrders": foodOrders,
	})
}

func GetAllFoodOrders(c *gin.Context) {
	var foodOrders []FoodOrder

	if err := DB.Order("created_at DESC").
		Limit(100).
		Find(&foodOrders).Error; err != nil {
		fmt.Printf("Error fetching all food orders: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch food orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"foodOrders": foodOrders,
	})
}

func GetRecentActivity(c *gin.Context) {
	var activities []Activity

	// Get recent income records
	var incomes []Income
	if err := DB.Preload("Guest").
		Order("created_at DESC").
		Limit(50).
		Find(&incomes).Error; err != nil {
		fmt.Printf("Error fetching recent income: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch recent activity"})
		return
	}

	// Get recent food orders
	var foodOrders []FoodOrder
	if err := DB.Order("created_at DESC").
		Limit(50).
		Find(&foodOrders).Error; err != nil {
		fmt.Printf("Error fetching recent food orders: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch recent activity"})
		return
	}

	// Combine activities
	for _, income := range incomes {
		activity := Activity{
			Type:        income.Type,
			Message:     fmt.Sprintf("%s Revenue", income.Type),
			Amount:      income.Amount,
			RoomNumber:  income.RoomNumber,
			Description: income.RevenueType,
			Timestamp:   income.CreatedAt,
		}
		if income.GuestID != nil {
			activity.GuestID = *income.GuestID
		}
		activities = append(activities, activity)
	}

	for _, order := range foodOrders {
		activities = append(activities, Activity{
			Type:       "food_order",
			Message:    fmt.Sprintf("Food order: %s (x%d)", order.FoodName, order.Quantity),
			Amount:     order.Price * float64(order.Quantity),
			GuestID:    order.GuestID,
			RoomNumber: int(order.RoomID),
			Timestamp:  order.CreatedAt,
		})
	}

	// Sort activities by timestamp (most recent first)
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].Timestamp.After(activities[j].Timestamp)
	})

	// Limit to most recent 50 activities
	if len(activities) > 50 {
		activities = activities[:50]
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
	})
}

func GetRevenueSummary(c *gin.Context) {
	today := time.Now().UTC().Format("2006-01-02")

	var totalRevenue, roomRevenue, foodRevenue, otherRevenue float64

	// Get room revenue
	if err := DB.Model(&Income{}).
		Where("DATE(created_at) = ? AND type = ?", today, "room").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&roomRevenue).Error; err != nil {
		fmt.Printf("Error getting room revenue: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to get revenue summary"})
		return
	}

	// Get food revenue
	if err := DB.Model(&Income{}).
		Where("DATE(created_at) = ? AND type = ?", today, "food").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&foodRevenue).Error; err != nil {
		fmt.Printf("Error getting food revenue: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to get revenue summary"})
		return
	}

	// Get other revenue
	if err := DB.Model(&Income{}).
		Where("DATE(created_at) = ? AND type = ?", today, "other").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&otherRevenue).Error; err != nil {
		fmt.Printf("Error getting other revenue: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to get revenue summary"})
		return
	}

	totalRevenue = roomRevenue + foodRevenue + otherRevenue

	c.JSON(http.StatusOK, gin.H{
		"totalRevenue": totalRevenue,
		"roomRevenue":  roomRevenue,
		"foodRevenue":  foodRevenue,
		"otherRevenue": otherRevenue,
		"date":         today,
	})
}

func GetRevenueRange(c *gin.Context) {
	startDate := c.Param("start")
	endDate := c.Param("end")

	var revenueData []RevenueData

	// Get daily revenue data for the date range using GORM
	err := DB.Model(&Income{}).
		Select(`DATE(created_at) as date,
			   SUM(CASE WHEN type = 'room' THEN amount ELSE 0 END) as room_revenue,
			   SUM(CASE WHEN type = 'food' THEN amount ELSE 0 END) as food_revenue,
			   SUM(CASE WHEN type = 'other' THEN amount ELSE 0 END) as other_revenue,
			   SUM(amount) as total_revenue`).
		Where("DATE(created_at) BETWEEN ? AND ?", startDate, endDate).
		Group("DATE(created_at)").
		Order("date").
		Scan(&struct {
			Date         string  `json:"date"`
			RoomRevenue  float64 `json:"roomRevenue"`
			FoodRevenue  float64 `json:"foodRevenue"`
			OtherRevenue float64 `json:"otherRevenue"`
			TotalRevenue float64 `json:"totalRevenue"`
		}{}).
		Find(&revenueData).Error

	if err != nil {
		fmt.Printf("Error getting revenue range: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch revenue data"})
		return
	}

	// Format dates in the response
	for i := range revenueData {
		if date, err := time.Parse("2006-01-02", revenueData[i].Date.Format("2006-01-02")); err == nil {
			revenueData[i].Date = date
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"revenueData": revenueData,
	})
}
