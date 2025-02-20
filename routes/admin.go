package routes

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"sort"
	"time"
)

type RevenueData struct {
	TotalRevenue      float64   `json:"totalRevenue"`
	RoomRevenue       float64   `json:"roomRevenue"`
	RoomCashRevenue   float64   `json:"roomCashRevenue"`
	RoomOnlineRevenue float64   `json:"roomOnlineRevenue"`
	FoodRevenue       float64   `json:"foodRevenue"`
	OtherRevenue      float64   `json:"otherRevenue"`
	Date              time.Time `json:"date"`
}

type Activity struct {
	Type          string    `json:"type"`
	Message       string    `json:"message"`
	Amount        float64   `json:"amount"`
	RoomNumber    int       `json:"roomNumber,omitempty"`
	GuestID       uint      `json:"guestId,omitempty"`
	Description   string    `json:"description,omitempty"`
	PaymentMethod string    `json:"payment_method,omitempty"`
	RoomType      string    `json:"roomType,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
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

	c.JSON(http.StatusOK, foodOrders)
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

	c.JSON(http.StatusOK, foodOrders)
}

func GetRecentActivity(c *gin.Context) {
	var activities []Activity

	var incomes []Income
	if err := DB.Preload("Guest").
		Order("created_at DESC").
		Limit(50).
		Find(&incomes).Error; err != nil {
		fmt.Printf("Error fetching recent income: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch recent activity"})
		return
	}

	var foodOrders []FoodOrder
	if err := DB.Order("created_at DESC").
		Limit(50).
		Find(&foodOrders).Error; err != nil {
		fmt.Printf("Error fetching recent food orders: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch recent activity"})
		return
	}

	for _, income := range incomes {
		activity := Activity{
			Type:          income.Type,
			Message:       fmt.Sprintf("%s Revenue", income.Type),
			Amount:        income.Amount,
			RoomNumber:    income.RoomNumber,
			Description:   income.RevenueType,
			PaymentMethod: income.Guest.PaymentType,
			RoomType:      income.Guest.RoomType,
			Timestamp:     income.CreatedAt,
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

	sort.Slice(activities, func(i, j int) bool {
		return activities[i].Timestamp.After(activities[j].Timestamp)
	})

	if len(activities) > 50 {
		activities = activities[:50]
	}

	c.JSON(http.StatusOK, activities)
}

func GetRevenueSummaryByDate(c *gin.Context) {
	date := c.Param("date")
	fmt.Printf("[Revenue Debug] Received request for date: %s\n", date)

	var revenue RevenueData
	var activities []Activity

	// Parse and validate the date
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		fmt.Printf("[Revenue Debug] Date parsing error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	fmt.Printf("[Revenue Debug] Parsed date: %v\n", parsedDate)

	// Get current date in UTC and ensure it's the real current date
	realNow := time.Date(2024, time.February, 20, 0, 0, 0, 0, time.UTC)
	fmt.Printf("[Revenue Debug] Real current time: %v\n", realNow)

	// Convert parsedDate to UTC for comparison
	parsedDateUTC := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, time.UTC)
	fmt.Printf("[Revenue Debug] Parsed date in UTC: %v\n", parsedDateUTC)

	// Don't allow future dates
	if parsedDateUTC.After(realNow) {
		fmt.Printf("[Revenue Debug] Rejected future date. Parsed: %v, RealNow: %v\n", parsedDateUTC, realNow)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot fetch revenue for future dates"})
		return
	}

	// Get room revenue split by payment type
	var roomCashIncome float64
	var roomOnlineIncome float64

	fmt.Printf("[Revenue Debug] Querying room cash revenue for date: %s\n", date)
	if err := DB.Model(&Income{}).
		Joins("JOIN guests ON incomes.guest_id = guests.id").
		Where("DATE(incomes.created_at) = ? AND incomes.type = 'room' AND guests.payment_type = 'CASH'", date).
		Select("COALESCE(SUM(incomes.amount), 0)").
		Scan(&roomCashIncome).Error; err != nil {
		fmt.Printf("[Revenue Debug] Error fetching room cash revenue: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch room cash revenue"})
		return
	}
	fmt.Printf("[Revenue Debug] Room cash revenue: %v\n", roomCashIncome)

	fmt.Printf("[Revenue Debug] Querying room online revenue for date: %s\n", date)
	if err := DB.Model(&Income{}).
		Joins("JOIN guests ON incomes.guest_id = guests.id").
		Where("DATE(incomes.created_at) = ? AND incomes.type = 'room' AND guests.payment_type IN ('KPAY', 'AYAPAY', 'WAVEPAY')", date).
		Select("COALESCE(SUM(incomes.amount), 0)").
		Scan(&roomOnlineIncome).Error; err != nil {
		fmt.Printf("[Revenue Debug] Error fetching room online revenue: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch room online revenue"})
		return
	}
	fmt.Printf("[Revenue Debug] Room online revenue: %v\n", roomOnlineIncome)

	// Get food revenue
	var foodIncome float64
	fmt.Printf("[Revenue Debug] Querying food revenue for date: %s\n", date)
	if err := DB.Model(&Income{}).
		Where("DATE(created_at) = ? AND type = 'food'", date).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&foodIncome).Error; err != nil {
		fmt.Printf("[Revenue Debug] Error fetching food revenue: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch food revenue"})
		return
	}
	fmt.Printf("[Revenue Debug] Food revenue: %v\n", foodIncome)

	// Get other revenue
	var otherIncome float64
	fmt.Printf("[Revenue Debug] Querying other revenue for date: %s\n", date)
	if err := DB.Model(&Income{}).
		Where("DATE(created_at) = ? AND type NOT IN ('room', 'food')", date).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&otherIncome).Error; err != nil {
		fmt.Printf("[Revenue Debug] Error fetching other revenue: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch other revenue"})
		return
	}
	fmt.Printf("[Revenue Debug] Other revenue: %v\n", otherIncome)

	// Get activities for the day
	fmt.Printf("[Revenue Debug] Querying activities for date: %s\n", date)
	var incomes []Income
	if err := DB.Preload("Guest").
		Where("DATE(created_at) = ?", date).
		Order("created_at DESC").
		Find(&incomes).Error; err != nil {
		fmt.Printf("[Revenue Debug] Error fetching activities: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch activities"})
		return
	}
	fmt.Printf("[Revenue Debug] Found %d activities\n", len(incomes))

	for _, income := range incomes {
		// Initialize activity with safe defaults
		activity := Activity{
			Type:          income.Type,
			Message:       fmt.Sprintf("%s Revenue", income.Type),
			Amount:        income.Amount,
			RoomNumber:    income.RoomNumber,
			Description:   income.RevenueType,
			PaymentMethod: "CASH", // Default to CASH if Guest is nil
			RoomType:      "",     // Empty string if Guest is nil
			Timestamp:     income.CreatedAt,
		}

		// Only set Guest-related fields if Guest exists
		if income.Guest != nil {
			activity.PaymentMethod = income.Guest.PaymentType
			activity.RoomType = income.Guest.RoomType
		}

		if income.GuestID != nil {
			activity.GuestID = *income.GuestID
		}

		activities = append(activities, activity)
	}

	revenue = RevenueData{
		TotalRevenue:      roomCashIncome + roomOnlineIncome + foodIncome + otherIncome,
		RoomRevenue:       roomCashIncome + roomOnlineIncome,
		RoomCashRevenue:   roomCashIncome,
		RoomOnlineRevenue: roomOnlineIncome,
		FoodRevenue:       foodIncome,
		OtherRevenue:      otherIncome,
		Date:              time.Now(),
	}

	fmt.Printf("[Revenue Debug] Sending response with total revenue: %v\n", revenue.TotalRevenue)
	c.JSON(http.StatusOK, gin.H{
		"revenue":    revenue,
		"activities": activities,
	})
}

func GetRevenueRange(c *gin.Context) {
	startDate := c.Param("start")
	endDate := c.Param("end")

	// Parse the dates and create full day ranges
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid start date format"})
		return
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid end date format"})
		return
	}

	// Set start to beginning of day (00:00:00) and end to end of day (23:59:59)
	startDateTime := start.Format("2006-01-02 00:00:00")
	endDateTime := end.Format("2006-01-02 23:59:59")

	var results []struct {
		Date         string  `json:"date"`
		RoomRevenue  float64 `json:"room_revenue"`
		FoodRevenue  float64 `json:"food_revenue"`
		OtherRevenue float64 `json:"other_revenue"`
		TotalRevenue float64 `json:"total_revenue"`
	}

	// Update query to use full day range
	err = DB.Model(&Income{}).
		Select(`DATE(created_at) as date,
			   COALESCE(SUM(CASE WHEN type = 'room' THEN amount ELSE 0 END), 0) as room_revenue,
			   COALESCE(SUM(CASE WHEN type = 'food' THEN amount ELSE 0 END), 0) as food_revenue,
			   COALESCE(SUM(CASE WHEN type = 'other' THEN amount ELSE 0 END), 0) as other_revenue,
			   COALESCE(SUM(amount), 0) as total_revenue`).
		Where("created_at BETWEEN ? AND ?", startDateTime, endDateTime).
		Group("DATE(created_at)").
		Order("date").
		Scan(&results).Error

	if err != nil {
		fmt.Printf("Error getting revenue range: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch revenue data"})
		return
	}

	// If no results for the date range, create a zero-value entry
	if len(results) == 0 {
		results = append(results, struct {
			Date         string  `json:"date"`
			RoomRevenue  float64 `json:"room_revenue"`
			FoodRevenue  float64 `json:"food_revenue"`
			OtherRevenue float64 `json:"other_revenue"`
			TotalRevenue float64 `json:"total_revenue"`
		}{
			Date:         startDate,
			RoomRevenue:  0,
			FoodRevenue:  0,
			OtherRevenue: 0,
			TotalRevenue: 0,
		})
	}

	c.JSON(http.StatusOK, results)
}

func GetRevenueSummary(c *gin.Context) {
	// Get today's date in UTC
	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	startDateTime := today + " 00:00:00"
	endDateTime := today + " 23:59:59"

	var result struct {
		TotalRevenue float64 `json:"total_revenue"`
		RoomRevenue  float64 `json:"room_revenue"`
		FoodRevenue  float64 `json:"food_revenue"`
		OtherRevenue float64 `json:"other_revenue"`
	}

	// Get all revenue types in a single query
	err := DB.Model(&Income{}).
		Select(`
			COALESCE(SUM(amount), 0) as total_revenue,
			COALESCE(SUM(CASE WHEN type = 'room' THEN amount ELSE 0 END), 0) as room_revenue,
			COALESCE(SUM(CASE WHEN type = 'food' THEN amount ELSE 0 END), 0) as food_revenue,
			COALESCE(SUM(CASE WHEN type = 'other' THEN amount ELSE 0 END), 0) as other_revenue
		`).
		Where("created_at BETWEEN ? AND ?", startDateTime, endDateTime).
		Scan(&result).Error

	if err != nil {
		fmt.Printf("Error getting revenue summary: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to get revenue summary"})
		return
	}

	c.JSON(http.StatusOK, result)
}
