package main

import (
	"AureoHMSBE/routes"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := "Aenlu:Hninhninlatt21!@tcp(87.106.203.188:3306)/Aureo_Cloud?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	routes.DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	dbError := routes.DB.AutoMigrate(
		&routes.Receptionist{},
		&routes.Admin{},
		&routes.Rooms{},
		&routes.Reservation{},
		&routes.Guests{},
		&routes.FoodOrder{},
		&routes.Menu{},
		&routes.Income{},
		&routes.Staff{},
		&routes.CleaningRecord{},
		&routes.DailyFoodRevenue{},
		&routes.RoomPrices{})
	if dbError != nil {
		return
	}
	fmt.Println("Database and tables created successfully")

	router := gin.Default()

	// Let Nginx handle CORS
	router.Use(func(c *gin.Context) {
		c.Next()
	})

	// Authentication
	router.POST("/login", routes.Login)
	router.POST("/admin/login", routes.AdminLogin)
	router.POST("/logout", routes.Logout)
	router.POST("/forgot-password", routes.ForgotPassword)
	router.POST("/staff-login", routes.StaffLogin)

	// Hotel website booking endpoint
	router.POST("/website-booking", routes.HandleWebsiteBooking)

	// Get routes
	router.GET("/check-auth", checkAuth)
	router.GET("/admin/check-auth", adminCheckAuth)

	// Reservation
	router.POST("/create-reservation", routes.CreateReservation)
	router.GET("reservations/date/:date", routes.GetReservationsByDate)
	router.GET("reservations/:id", routes.GetReservation)
	router.DELETE("reservations/:id", routes.DeleteReservation)
	router.PUT("reservations/:id", routes.UpdateReservation)

	//Rooms
	router.GET("/rooms", routes.GetRooms)
	router.GET("/rooms/:room", routes.GetRoom)
	router.PUT("rooms/:room", routes.UpdateRoomStatus)

	// Room Prices
	router.GET("/prices", routes.GetRoomPrices)
	router.POST("/prices", routes.UpdateRoomPrices)

	//Guests
	router.POST("/create-guest", routes.CreateGuest)
	router.GET("/guests/current/:roomNumber", routes.GetCurrentGuest)
	router.GET("/guests/checkouts/today", routes.GetTodayCheckouts)
	router.PUT("/guests/:id", routes.UpdateGuestInfo)
	router.PUT("/guests/foodPrice/:id", routes.UpdateGuestFoodPrice)

	// Food
	router.POST("/food/order", routes.CreateFoodOrder)
	router.GET("/food/order/:id", routes.GetFoodOrder)
	router.GET("/food/orders/:roomId", routes.GetFoodOrdersByRoom)
	router.GET("/food/orders/guest/:guestId", routes.GetFoodOrdersByGuestID)
	router.GET("/food/revenue/today", routes.GetTodayFoodRevenue)
	router.GET("/food/revenue/date/:date", routes.GetFoodRevenueByDate)
	router.PUT("/order/:id", routes.UpdateFoodOrder)
	router.DELETE("/order/:id", routes.DeleteFoodOrder)

	router.POST("/food/menu", routes.CreateMenu)
	router.GET("food/menus", routes.GetMenu)
	router.GET("food/menu/:id", routes.GetMenuByID)
	router.GET("food/menus/:foodName", routes.GetMenuByName)
	router.GET("food/search", routes.SearchMenu)
	router.PUT("/menu/:id", routes.UpdateMenu)
	router.DELETE("/menu/:id", routes.DeleteMenu)

	// Income Record
	router.POST("/income", routes.AddIncome)
	router.GET("income/today", routes.GetTodayIncome)
	router.GET("income/date/:date", routes.GetIncomeByDate)

	// Admin routes without protection
	router.GET("/admin/revenue", routes.GetRevenue)
	router.GET("/admin/revenue/date/:date", routes.GetRevenueByDate)
	router.GET("/admin/revenue/month/:month", routes.GetRevenueByMonth)
	router.GET("/admin/revenue/year/:year", routes.GetRevenueByYear)

	// Food orders
	router.GET("/admin/food-orders/date/:date", routes.GetFoodOrdersByDate)
	router.GET("/admin/food-orders/all", routes.GetAllFoodOrders)

	// Recent activity
	router.GET("/admin/activity", routes.GetRecentActivity)

	// Revenue data
	router.GET("/admin/revenue/summary", routes.GetRevenueSummary)
	router.GET("/admin/revenue/range/:start/:end", routes.GetRevenueRange)

	// Protected routes group
	protected := router.Group("/")
	protected.Use(routes.AuthMiddleware())

	// Add your protected routes here
	protected.GET("/stats", routes.GetDashboardStats)

	// Start the server
	fmt.Println("Server starting on :8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}

func checkAuth(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Not logged in"})
		return
	}

	// The AuthMiddleware will handle token validation
	routes.AuthMiddleware()(c)
}

func adminCheckAuth(c *gin.Context) {
	routes.AdminCheckAuth(c)
}
