package routes

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type FoodOrder struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	GuestID   uint      `gorm:"not null"`
	RoomID    uint      `gorm:"not null"`
	FoodName  string    `gorm:"not null"`
	Price     float64   `gorm:"not null"`
	Quantity  uint      `gorm:"not null"`
	OrderTime time.Time `gorm:"type:datetime;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type Menu struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	FoodName  string `gorm:"not null"`
	FoodPrice string `gorm:"not null"`
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

	if order.OrderTime.IsZero() {
		order.OrderTime = time.Now()
	}

	if err := DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create food order: " + err.Error()})
		return
	}

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

	if err := DB.Where("room_id = ?", roomID).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch food orders"})
		return
	}

	c.JSON(http.StatusOK, orders)
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