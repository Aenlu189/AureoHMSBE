package routes

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

// GetActivitiesByDate retrieves all income activities for a specific date
func GetActivitiesByDate(db *gorm.DB, date time.Time) ([]Income, error) {
	var activities []Income

	// Set the time range for the given date
	startTime := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endTime := startTime.Add(24 * time.Hour)

	// Query the database
	err := db.Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Order("created_at desc").
		Find(&activities).Error

	return activities, err
}

// GetRevenueByDate handles GET /admin/revenue/date/:date
func GetRevenueByDate(c *gin.Context) {
	date := c.Param("date")
	fmt.Printf("Fetching revenue for date: %s\n", date) // Debug log
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Date parameter is required"})
		return
	}

	// Parse the date
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	// Get activities for the date
	activities, err := GetActivitiesByDate(DB, parsedDate)
	fmt.Printf("Found %d activities\n", len(activities)) // Debug log
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch activities"})
		return
	}

	// Debug log activity data
	for i, activity := range activities {
		fmt.Printf("Activity %d: Type=%s, Amount=%f\n", i, activity.Type, activity.Amount)
	}

	// Format the response
	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
		"date":       date,
		"success":    true,
	})
}

func isOnlinePayment(method string) bool {
	onlinePayments := []string{"KPAY", "WAVEPAY", "AYAPAY", "CBPAY"}
	for _, p := range onlinePayments {
		if p == method {
			return true
		}
	}
	return false
}

// GetRevenue handles GET /admin/revenue
func GetRevenue(c *gin.Context) {
	var result struct {
		TotalRevenue      float64 `json:"totalRevenue"`
		RoomCashRevenue   float64 `json:"roomCashRevenue"`
		RoomOnlineRevenue float64 `json:"roomOnlineRevenue"`
		FoodRevenue       float64 `json:"foodRevenue"`
		OtherRevenue      float64 `json:"otherRevenue"`
	}

	// Get all revenue
	var activities []Income
	if err := DB.Find(&activities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch revenue data"})
		return
	}

	// Calculate revenues
	for _, activity := range activities {
		amount := activity.Amount

		switch activity.Type {
		case "CHECKED-IN", "EXTEND-STAY":
			if isOnlinePayment(activity.PaymentMethod) {
				result.RoomOnlineRevenue += amount
			} else {
				result.RoomCashRevenue += amount
			}
		case "FOOD", "EMPLOYEE_FOOD", "GUEST_FOOD":
			result.FoodRevenue += amount
		default:
			result.OtherRevenue += amount
		}
	}

	result.TotalRevenue = result.RoomCashRevenue + result.RoomOnlineRevenue +
		result.FoodRevenue + result.OtherRevenue

	c.JSON(http.StatusOK, result)
}

// GetRevenueByMonth handles GET /admin/revenue/month/:month
func GetRevenueByMonth(c *gin.Context) {
	month := c.Param("month")
	if month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Month parameter is required"})
		return
	}

	// Parse the month (expected format: "2024-02")
	parsedDate, err := time.Parse("2006-01", month)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month format. Use YYYY-MM"})
		return
	}

	// Calculate start and end of month
	startOfMonth := time.Date(parsedDate.Year(), parsedDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

	var result struct {
		Activities        []Income `json:"activities"`
		TotalRevenue      float64  `json:"totalRevenue"`
		RoomCashRevenue   float64  `json:"roomCashRevenue"`
		RoomOnlineRevenue float64  `json:"roomOnlineRevenue"`
		FoodRevenue       float64  `json:"foodRevenue"`
		OtherRevenue      float64  `json:"otherRevenue"`
	}

	// Get activities for the month
	if err := DB.Where("created_at BETWEEN ? AND ?", startOfMonth, endOfMonth).
		Order("created_at desc").
		Find(&result.Activities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch activities"})
		return
	}

	// Calculate revenues
	for _, activity := range result.Activities {
		amount := activity.Amount

		switch activity.Type {
		case "CHECKED-IN", "EXTEND-STAY":
			if isOnlinePayment(activity.PaymentMethod) {
				result.RoomOnlineRevenue += amount
			} else {
				result.RoomCashRevenue += amount
			}
		case "FOOD", "EMPLOYEE_FOOD", "GUEST_FOOD":
			result.FoodRevenue += amount
		default:
			result.OtherRevenue += amount
		}
	}

	result.TotalRevenue = result.RoomCashRevenue + result.RoomOnlineRevenue +
		result.FoodRevenue + result.OtherRevenue

	c.JSON(http.StatusOK, result)
}

// GetRevenueByYear handles GET /admin/revenue/year/:year
func GetRevenueByYear(c *gin.Context) {
	year := c.Param("year")
	if year == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Year parameter is required"})
		return
	}

	// Parse the year
	parsedYear, err := time.Parse("2006", year)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid year format. Use YYYY"})
		return
	}

	// Calculate start and end of year
	startOfYear := time.Date(parsedYear.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	endOfYear := startOfYear.AddDate(1, 0, 0).Add(-time.Second)

	var result struct {
		Activities        []Income `json:"activities"`
		TotalRevenue      float64  `json:"totalRevenue"`
		RoomCashRevenue   float64  `json:"roomCashRevenue"`
		RoomOnlineRevenue float64  `json:"roomOnlineRevenue"`
		FoodRevenue       float64  `json:"foodRevenue"`
		OtherRevenue      float64  `json:"otherRevenue"`
	}

	// Get activities for the year
	if err := DB.Where("created_at BETWEEN ? AND ?", startOfYear, endOfYear).
		Order("created_at desc").
		Find(&result.Activities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch activities"})
		return
	}

	// Calculate revenues
	for _, activity := range result.Activities {
		amount := activity.Amount

		switch activity.Type {
		case "CHECKED-IN", "EXTEND-STAY":
			if isOnlinePayment(activity.PaymentMethod) {
				result.RoomOnlineRevenue += amount
			} else {
				result.RoomCashRevenue += amount
			}
		case "FOOD", "EMPLOYEE_FOOD", "GUEST_FOOD":
			result.FoodRevenue += amount
		default:
			result.OtherRevenue += amount
		}
	}

	result.TotalRevenue = result.RoomCashRevenue + result.RoomOnlineRevenue +
		result.FoodRevenue + result.OtherRevenue

	c.JSON(http.StatusOK, result)
}
