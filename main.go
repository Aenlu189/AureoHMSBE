package main

import (
	"AureoHMSBE/routes"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
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
		&routes.CleaningRecord{})
	if dbError != nil {
		return
	}
	fmt.Println("Database and tables created successfully")

	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://localhost:63343", // Local development
		"http://127.0.0.1:63343", // For local development
		"http://87.106.203.188",  // VPS IP
		"http://aureocloud.co.uk",
		"http://www.aureocloud.co.uk",
	}
	config.AllowCredentials = true
	config.AllowHeaders = []string{
		"Origin",
		"Content-Length",
		"Content-Type",
		"Authorization",
		"Accept",
		"Cookie",
	}
	config.ExposeHeaders = []string{"Set-Cookie"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	router.Use(cors.New(config))

	// Create a new cookie store with a secure key
	store := cookie.NewStore([]byte("AureoHMS-Session-Key-2025-01-23"))

	// Configure session middleware
	router.Use(sessions.Sessions("mysession", store))

	// Add session middleware with appropriate settings
	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		origin := c.GetHeader("Origin")
		fmt.Printf("Request Origin: %s\n", origin)

		options := sessions.Options{
			Path:     "/",
			MaxAge:   3600 * 24,
			HttpOnly: true,
			Secure:   false,                // Set to false since we're not using HTTPS yet
			Domain:   "",                   // Empty for now to work with IP
			SameSite: http.SameSiteLaxMode, // Less restrictive for cross-domain
		}

		session.Options(options)
		c.Next()
	})

	// Authentication
	router.POST("/login", routes.Login)
	router.POST("/logout", routes.Logout)
	router.GET("/check-session", func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")
		fmt.Printf("Check Session - Current user: %v\n", user)
		fmt.Printf("Request Headers: %v\n", c.Request.Header)

		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Not logged in"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Logged in", "user": user})
	})

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

	// Staff routes
	router.POST("/staff/login", routes.StaffLogin)

	// Protected staff routes
	staffRoutes := router.Group("/staff")
	staffRoutes.Use(routes.StaffAuthMiddleware())
	{
		staffRoutes.GET("/rooms", routes.GetRoomsForCleaning)
		staffRoutes.POST("/cleaning/start", routes.StartCleaning)
		staffRoutes.POST("/cleaning/complete", routes.CompleteCleaning)
		staffRoutes.GET("/history", routes.GetStaffCleaningHistory)
	}

	protected := router.Group("/")
	protected.Use(routes.AuthMiddleware())

	protected.GET("/stats", routes.GetDashboardStats)

	runErr := router.Run(":8080")
	if runErr != nil {
		fmt.Printf("Server failed to start: %v\n", runErr)
		return
	}
}
