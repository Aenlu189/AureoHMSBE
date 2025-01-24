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
	dsn := "root:Hninhninlatt21@tcp(127.0.0.1:3306)/Aureo_Cloud?charset=utf8mb4&parseTime=True&loc=Local"
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
	config.AllowOrigins = []string{"http://aureocloud.co.uk"}
	config.AllowCredentials = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.ExposeHeaders = []string{"Set-Cookie"}
	router.Use(cors.New(config))

	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 24,
		HttpOnly: false,
		Secure:   false,
		Domain:   "aureocloud.co.uk",
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("mysession", store))

	// Authentication
	router.POST("/login", routes.Login)
	router.POST("/logout", routes.Logout)
	router.POST("/forgot-password", routes.ForgotPassword)
	router.POST("/admin", routes.AdminLogin)
	router.GET("/check-session", checkSession)

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

	protected := router.Group("/")
	protected.Use(routes.AuthMiddleware())

	protected.GET("/stats", routes.GetDashboardStats)

	runErr := router.Run(":8080")
	if runErr != nil {
		fmt.Printf("Localhost server not running")
		return
	}
}

func checkSession(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	fmt.Println("Session user: ", user)

	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Not logged in"})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": user})
	}
}
