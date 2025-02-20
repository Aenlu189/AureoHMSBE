package routes

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

type FoodOrder struct {
	ID            uint      `gorm:"primaryKey;autoIncrement"`
	GuestID       uint      `gorm:"not null"`
	RoomID        uint      `gorm:"not null"`
	FoodName      string    `gorm:"not null"`
	Price         float64   `gorm:"not null"`
	Quantity      uint      `gorm:"not null"`
	OrderTime     time.Time `gorm:"type:datetime;not null"`
	PaymentMethod string    `gorm:"type:varchar(10);default:'CASH'"`
	PaymentStatus string    `gorm:"type:varchar(20);default:'PENDING'"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

func (order *FoodOrder) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().UTC()
	order.CreatedAt = now
	order.UpdatedAt = now
	if order.OrderTime.IsZero() {
		order.OrderTime = now
	}
	return nil
}

func (order *FoodOrder) BeforeUpdate(tx *gorm.DB) error {
	order.UpdatedAt = time.Now().UTC()
	return nil
}

type Menu struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	FoodName  string `gorm:"not null"`
	FoodPrice string `gorm:"not null"`
}

type DailyFoodRevenue struct {
	ID            uint      `gorm:"primaryKey;autoIncrement"`
	Date          time.Time `gorm:"type:date;uniqueIndex:idx_date_payment"`
	PaymentMethod string    `gorm:"type:varchar(10);not null;uniqueIndex:idx_date_payment"`
	Revenue       float64   `gorm:"not null;default:0"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

// Validate payment method
func isValidPaymentMethod(method string) bool {
	validMethods := []string{"CASH", "KPAY", "AYAPAY", "WAVEPAY"}
	method = strings.ToUpper(method)
	for _, valid := range validMethods {
		if method == valid {
			return true
		}
	}
	return false
}

func CreateMenu(c *gin.Context) {
	var menu Menu

	if err := c.BindJSON(&menu); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	if err := DB.Create(&menu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create menu: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":   "Food menu created successsfully",
		"Food menu": menu,
	})
}

func GetMenu(c *gin.Context) {
	var menu []Menu
	if err := DB.Find(&menu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, menu)
}

func GetMenuByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Menu ID is required"})
		return
	}

	var menu Menu
	if err := DB.Find(&menu, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Food not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, menu)
}

func GetMenuByName(c *gin.Context) {
	name := c.Param("food_name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Food name is required"})
		return
	}

	var menu Menu
	if err := DB.Find(&menu, name).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Food not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, menu)
}

func UpdateMenu(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Menu ID is required"})
		return
	}

	var menu Menu
	if err := c.BindJSON(&menu); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	var existingMenu Menu
	if err := DB.First(&existingMenu, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Menu not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	if err := DB.Model(&existingMenu).Updates(menu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existingMenu)
}

func DeleteMenu(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Menu ID is not found"})
		return
	}

	var menu []Menu
	if err := DB.First(&menu, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Menu not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	if err := DB.Delete(&menu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete the menu"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Menu deleted successfully"})
}

func CreateFoodOrder(c *gin.Context) {
	var order FoodOrder

	if err := c.BindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// Validate payment method
	if order.PaymentMethod == "" {
		order.PaymentMethod = "CASH"
	} else if !isValidPaymentMethod(order.PaymentMethod) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid payment method"})
		return
	}

	// Set initial payment status
	if order.PaymentMethod == "CASH" {
		order.PaymentStatus = "PAID"
	} else {
		order.PaymentStatus = "PAID"
	}

	// Start a transaction
	tx := DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to start transaction"})
		return
	}

	// Create the food order
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create food order: " + err.Error()})
		return
	}

	// Record revenue by payment method
	today := time.Now().UTC().Truncate(24 * time.Hour)
	var revenue DailyFoodRevenue
	result := tx.Where("date = ? AND payment_method = ?", today, order.PaymentMethod).First(&revenue)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new revenue record
			revenue = DailyFoodRevenue{
				Date:          today,
				PaymentMethod: order.PaymentMethod,
				Revenue:       order.Price * float64(order.Quantity),
			}
			if err := tx.Create(&revenue).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create revenue record"})
				return
			}
		} else {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to check revenue record"})
			return
		}
	} else {
		// Update existing revenue record
		revenue.Revenue += order.Price * float64(order.Quantity)
		if err := tx.Save(&revenue).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update revenue record"})
			return
		}
	}

	// Only update guest's food charges for cash payments
	if order.PaymentMethod == "CASH" {
		var guest Guests
		if err := tx.First(&guest, "room_number = ?", order.RoomID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update guest food charges"})
			return
		}

		guest.FoodCharges += int(order.Price * float64(order.Quantity))
		if err := tx.Save(&guest).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update guest food charges"})
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Food order created successfully",
		"order":   order,
	})
}

func GetFoodOrder(c *gin.Context) {
	id := c.Param("id")
	var order FoodOrder

	if err := DB.First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "Food order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch food order"})
		return
	}

	c.JSON(http.StatusOK, order)
}

func GetFoodOrdersByRoom(c *gin.Context) {
	roomID := c.Param("roomId")
	var orders []FoodOrder

	if err := DB.Where("room_id = ?", roomID).
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch food orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"summary": map[string]interface{}{
			"total_orders":     len(orders),
			"pending_payments": countPendingPayments(orders),
			"total_amount":     calculateTotalAmount(orders),
		},
	})
}

func GetFoodOrdersByGuestID(c *gin.Context) {
	guestID := c.Param("guestId")

	var orders []FoodOrder
	if err := DB.Where("guest_id = ?", guestID).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch food orders"})
		return
	}

	c.JSON(http.StatusOK, orders)
}

func UpdateFoodOrder(c *gin.Context) {
	id := c.Param("id")
	var order FoodOrder

	if err := DB.First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "Food order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch food order"})
		return
	}

	var updateData struct {
		Status string `json:"status"`
	}

	if err := c.BindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Food order updated successfully",
		"order":   order,
	})
}

func DeleteFoodOrder(c *gin.Context) {
	id := c.Param("id")
	var order FoodOrder

	if err := DB.First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "Food order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch food order"})
		return
	}

	// Update guest's food charges before deleting the order
	var guest Guests
	if err := DB.First(&guest, "room_number = ?", order.RoomID).Error; err == nil {
		guest.FoodCharges -= int(order.Price)
		DB.Save(&guest)
	}

	if err := DB.Delete(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete food order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Food order deleted successfully"})
}

func GetDailyFoodRevenue() float64 {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var foodOrders []FoodOrder
	if err := DB.Where("DATE(order_time) = ?", today.Format("2006-01-02")).Find(&foodOrders).Error; err != nil {
		return 0
	}

	var totalRevenue float64
	for _, order := range foodOrders {
		totalRevenue += order.Price * float64(order.Quantity)
	}
	return totalRevenue
}

func GetTodayFoodRevenue(c *gin.Context) {
	var totalRevenue float64
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if err := DB.Model(&FoodOrder{}).
		Where("DATE(order_time) = ?", today).
		Select("COALESCE(SUM(price * quantity), 0)").
		Scan(&totalRevenue).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to calculate today's food revenue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"foodRevenue": totalRevenue})
}

func GetFoodRevenueByDate(c *gin.Context) {
	date := c.Param("date")
	var totalRevenue float64

	if err := DB.Model(&FoodOrder{}).
		Where("DATE(order_time) = ?", date).
		Select("COALESCE(SUM(price * quantity), 0)").
		Scan(&totalRevenue).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to calculate food revenue for the date"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"foodRevenue": totalRevenue})
}

func SearchMenu(c *gin.Context) {
	searchTerm := c.Query("term")
	var menus []Menu

	if searchTerm != "" {
		if err := DB.Where("food_name LIKE ?", "%"+searchTerm+"%").Find(&menus).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to search menu items"})
			return
		}
	} else {
		if err := DB.Find(&menus).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch menu items"})
			return
		}
	}

	c.JSON(http.StatusOK, menus)
}

// New function to handle payment confirmation
func UpdateFoodOrderPayment(c *gin.Context) {
	id := c.Param("id")
	var order FoodOrder

	if err := DB.First(&order, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Food order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch food order"})
		return
	}

	var paymentUpdate struct {
		PaymentStatus string `json:"payment_status"`
	}

	if err := c.BindJSON(&paymentUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// Validate payment status
	if !isValidPaymentStatus(paymentUpdate.PaymentStatus) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid payment status"})
		return
	}

	// If payment is confirmed, update guest's food charges
	if paymentUpdate.PaymentStatus == "PAID" && order.PaymentStatus != "PAID" {
		var guest Guests
		if err := DB.First(&guest, "room_number = ?", order.RoomID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update guest food charges"})
			return
		}

		guest.FoodCharges += int(order.Price)
		if err := DB.Save(&guest).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update guest food charges"})
			return
		}
	}

	// Update order payment status
	order.PaymentStatus = paymentUpdate.PaymentStatus
	if err := DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update payment status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment status updated successfully",
		"order":   order,
	})
}

// Helper function to validate payment status
func isValidPaymentStatus(status string) bool {
	validStatuses := []string{"PENDING", "PAID", "FAILED", "CANCELLED"}
	status = strings.ToUpper(status)
	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// Helper functions for order summary
func countPendingPayments(orders []FoodOrder) int {
	count := 0
	for _, order := range orders {
		if order.PaymentStatus == "PENDING" {
			count++
		}
	}
	return count
}

func calculateTotalAmount(orders []FoodOrder) float64 {
	var total float64
	for _, order := range orders {
		if order.PaymentStatus == "PAID" {
			total += order.Price * float64(order.Quantity)
		}
	}
	return total
}
