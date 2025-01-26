package main

import (
	"AureoHMSBE/routes"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
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
		&routes.DailyFoodRevenue{})
	if dbError != nil {
		return
	}
	fmt.Println("Database and tables created successfully")

	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://aureocloud.co.uk", "http://127.0.0.1:5500"}
	config.AllowCredentials = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	router.Use(cors.New(config))

	// Authentication
	router.POST("/login", routes.Login)
	router.POST("/logout", routes.Logout)
	router.POST("/forgot-password", routes.ForgotPassword)
	router.POST("/admin", routes.AdminLogin)
	router.GET("/check-auth", checkAuth)

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
	router.PUT("/menu/:id", routes.UpdateMenu)
	router.DELETE("/menu/:id", routes.DeleteMenu)

	// Income Record
	router.POST("/income", routes.AddIncome)
	router.GET("income/today", routes.GetTodayIncome)
	router.GET("income/date/:date", routes.GetIncomeByDate)

	// Protected routes group
	protected := router.Group("/")
	protected.Use(routes.AuthMiddleware())

	// Add your protected routes here
	protected.GET("/stats", routes.GetDashboardStats)

	fmt.Println("Server is running on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
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
