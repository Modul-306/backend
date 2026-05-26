package api

import (
	"fmt"
	"log"
)

type NotificationService interface {
	SendEmail(to, subject, body string) error
}

type MockEmailService struct{}

func (m *MockEmailService) SendEmail(to, subject, body string) error {
	log.Printf("[EMAIL MOCK] To: %s | Subject: %s | Body preview: %s...", to, subject, body[:min(len(body), 50)])
	fmt.Printf("\n--- NEW EMAIL ---\nTo: %s\nSubject: %s\nBody: %s\n------------------\n", to, subject, body)
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
