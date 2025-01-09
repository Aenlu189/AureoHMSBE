package routes

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
	"log"
	"net/http"
)

func ForgotPassword(c *gin.Context) {
	var requestData struct {
		Email string `json:"email"`
	}

	if err := c.BindJSON(&requestData); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	var user Receptionist
	result := DB.Where("email = ?", requestData.Email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Invalid email"})
		} else {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": result.Error.Error()})
		}
		return
	}

	subject := "Password Recovery - Aureo Cloud"
	body := fmt.Sprintf("Hello %s,\n\nYour password is: %s\n\nBest regards,\nAen", user.Name, user.Password)
	if err := sendEmail(requestData.Email, subject, body); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Failed to send email"})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"message": "Password is sent to your email"})
}

func sendEmail(to, subject, body string) error {
	from := "aureo.yangon@gmail.com"
	password := "cfwcbkgsfvnntsav"
	smtpHost := "smtp.gmail.com"
	smtpPort := 587

	mailer := gomail.NewMessage()
	mailer.SetHeader("From", from)
	mailer.SetHeader("To", to)
	mailer.SetHeader("Subject", subject)
	mailer.SetBody("text/plain", body)

	dialer := gomail.NewDialer(smtpHost, smtpPort, from, password)

	if err := dialer.DialAndSend(mailer); err != nil {
		log.Printf("Could not send an emal: %v", err)
		return err
	}
	fmt.Println("Email sent successfully")
	return nil
}
