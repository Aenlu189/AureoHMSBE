package main

import (
	"AureoHMSBE/routes"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"net/http"
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
		&routes.Guests{})
	if dbError != nil {
		return
	}
	fmt.Println("Database and tables created successfully")

	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:63343"}
	config.AllowCredentials = true
	config.AllowHeaders = []string{"Origin", "Content-length", "Content-Type", "Authorization"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	router.Use(cors.New(config))

	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 24,
		HttpOnly: true,
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
	router.POST("/create-guest", routes.CreateGuest)
	router.GET("/rooms", routes.GetRooms)
	router.GET("/rooms/:room", routes.GetRoom)
	router.PUT("rooms/:room", routes.UpdateRoomStatus)

	protected := router.Group("/")
	protected.Use(routes.AuthMiddleware())

	protected.GET("/stats", routes.GetDashboardStats)

	runErr := router.Run("localhost:8080")
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
