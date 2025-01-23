package main

import (
	"AureoHMSBE/routes"
	"fmt"
	"log"
	"net/http"
	"os"

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

	// Add CORS middleware before any routes
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Allow all origins for debugging
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60,
	}))

	// Setup session store with minimal options
	store := cookie.NewStore([]byte("your-secret-key"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	// Use session middleware
	router.Use(sessions.Sessions("mysession", store))

	// Add logging middleware
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Output: os.Stdout,
		Formatter: func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %s\n",
				param.TimeStamp.Format("2006/01/02 - 15:04:05"),
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				param.Path,
			)
		},
	}))

	// Public routes
	router.POST("/login", routes.Login)
	router.POST("/logout", routes.Logout)
	router.GET("/check-session", routes.CheckAuth)

	// Protected routes
	authorized := router.Group("/")
	authorized.Use(routes.AuthMiddleware())
	{
		// User Management
		authorized.POST("/update-password", routes.UpdatePassword)

		// Stats
		authorized.GET("/stats", routes.GetDashboardStats)

		// Reservations
		authorized.POST("/create-reservation", routes.CreateReservation)
		authorized.GET("/reservations/date/:date", routes.GetReservationsByDate)
		authorized.GET("/reservations/:id", routes.GetReservation)
		authorized.PUT("/reservations/:id", routes.UpdateReservation)
		authorized.DELETE("/reservations/:id", routes.DeleteReservation)

		// Rooms
		authorized.GET("/rooms", routes.GetRooms)
		authorized.GET("/rooms/:room", routes.GetRoom)
		authorized.PUT("/rooms/:room", routes.UpdateRoomStatus)

		// Guests
		authorized.POST("/create-guest", routes.CreateGuest)
		authorized.GET("/guests/current/:roomNumber", routes.GetCurrentGuest)
		authorized.GET("/guests/checkouts/today", routes.GetTodayCheckouts)
		authorized.PUT("/guests/:id", routes.UpdateGuestInfo)
		authorized.PUT("/guests/foodPrice/:id", routes.UpdateGuestFoodPrice)

		// Food Orders
		authorized.POST("/food/order", routes.CreateFoodOrder)
		authorized.GET("/food/order/:id", routes.GetFoodOrder)
		authorized.GET("/food/orders/:roomId", routes.GetFoodOrdersByRoom)
		authorized.GET("/food/orders/guest/:guestId", routes.GetFoodOrdersByGuestID)
		authorized.PUT("/order/:id", routes.UpdateFoodOrder)
		authorized.DELETE("/order/:id", routes.DeleteFoodOrder)

		// Menu
		authorized.POST("/food/menu", routes.CreateMenu)
		authorized.GET("/food/menus", routes.GetMenu)
		authorized.GET("/food/menu/:id", routes.GetMenuByID)
		authorized.GET("/food/menus/:foodName", routes.GetMenuByName)
		authorized.PUT("/menu/:id", routes.UpdateMenu)
		authorized.DELETE("/menu/:id", routes.DeleteMenu)

		// Income
		authorized.POST("/income", routes.AddIncome)
		authorized.GET("/income/today", routes.GetTodayIncome)
		authorized.GET("/income/date/:date", routes.GetIncomeByDate)

		// Staff
		authorized.POST("/staff/login", routes.StaffLogin)
		authorized.GET("/staff/rooms", routes.GetRoomsForCleaning)
		authorized.POST("/staff/cleaning/start", routes.StartCleaning)
		authorized.POST("/staff/cleaning/complete", routes.CompleteCleaning)
		authorized.GET("/staff/history", routes.GetStaffCleaningHistory)
	}

	err = router.Run(":8080")
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
