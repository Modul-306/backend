package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/M306/backend/internal/db/sqlc"
	"github.com/google/uuid"
)

type payrexxWebhookPayload struct {
	ID          int    `json:"id"`
	Status      string `json:"status"`
	ReferenceID string `json:"referenceId"`
	Transaction *struct {
		ID          int    `json:"id"`
		Status      string `json:"status"`
		ReferenceID string `json:"referenceId"`
	} `json:"transaction"`
}

func (s *Server) handlePayrexxWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the raw body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Failed to read request body")
		return
	}

	// Verify the webhook signature
	signature := r.Header.Get("X-Webhook-Signature")
	if signature == "" {
		s.errorResponse(w, r, http.StatusUnauthorized, "Missing webhook signature")
		return
	}

	// Compute local signature
	mac := hmac.New(sha256.New, []byte(s.payrexxAPISecret))
	mac.Write(bodyBytes)
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		s.errorResponse(w, r, http.StatusUnauthorized, "Invalid webhook signature")
		log.Printf("Payrexx Webhook: Signature verification failed. Expected: %s, Got: %s", expectedSignature, signature)
		return
	}

	var txStatus string
	var txRefID string

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		// Parse JSON payload
		var payload payrexxWebhookPayload
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			s.errorResponse(w, r, http.StatusBadRequest, "Invalid JSON payload")
			return
		}
		txStatus = payload.Status
		txRefID = payload.ReferenceID
		if payload.Transaction != nil {
			if txStatus == "" {
				txStatus = payload.Transaction.Status
			}
			if txRefID == "" {
				txRefID = payload.Transaction.ReferenceID
			}
		}
	} else {
		// Parse as form urlencoded
		values, err := url.ParseQuery(string(bodyBytes))
		if err != nil {
			s.errorResponse(w, r, http.StatusBadRequest, "Failed to parse form urlencoded payload")
			return
		}
		txStatus = values.Get("transaction[status]")
		txRefID = values.Get("transaction[referenceId]")
		if txStatus == "" {
			txStatus = values.Get("status")
		}
		if txRefID == "" {
			txRefID = values.Get("referenceId")
		}
	}

	if txRefID == "" {
		s.errorResponse(w, r, http.StatusBadRequest, "Missing referenceId")
		return
	}

	orderID, err := uuid.Parse(txRefID)
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid referenceId UUID format")
		return
	}

	// Retrieve order from database
	order, err := s.db.GetOrder(r.Context(), orderID)
	if err != nil {
		s.errorResponse(w, r, http.StatusNotFound, "Order not found")
		return
	}

	// Process transaction status
	var newStatus string
	var newPaymentStatus string

	switch txStatus {
	case "confirmed":
		newStatus = "completed"
		newPaymentStatus = "paid"
	case "cancelled", "declined", "error":
		newStatus = "cancelled"
		newPaymentStatus = "unpaid"
	default:
		// Other statuses (like waiting, authorized) don't change the order state yet
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ignored"}`))
		return
	}

	// Update order status and payment status in database
	_, err = s.db.UpdateOrderPaymentStatus(r.Context(), db.UpdateOrderPaymentStatusParams{
		ID:            order.ID,
		Status:        newStatus,
		PaymentStatus: newPaymentStatus,
	})
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to update order status")
		return
	}

	// Send confirmation/cancellation email asynchronously
	go func() {
		u, err := s.db.GetUserByID(r.Context(), order.UserID)
		if err == nil {
			if newStatus == "completed" {
				_ = s.notifications.SendEmail(u.Email, "Order Payment Confirmed", "Your order payment has been successfully processed! Your order status is now completed.")
			} else if newStatus == "cancelled" {
				_ = s.notifications.SendEmail(u.Email, "Order Payment Failed", "Your order payment has failed or was cancelled. Your order status is now cancelled.")
			}
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

func (s *Server) createPayrexxGateway(orderID uuid.UUID, totalAmount float64) (int, string, error) {
	// Payrexx amounts are in cents. Convert totalAmount (float64) to cents (int).
	amountInCents := int(totalAmount * 100)

	// API Endpoint
	apiURL := fmt.Sprintf("https://api.payrexx.com/v1.0/Gateway/?instance=%s", s.payrexxInstance)

	// Determine frontend URL
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	// Construct parameters
	params := url.Values{}
	params.Set("amount", fmt.Sprintf("%d", amountInCents))
	params.Set("currency", "CHF")
	params.Set("referenceId", orderID.String())
	params.Set("successRedirectUrl", fmt.Sprintf("%s/orders/%s", frontendURL, orderID.String()))
	params.Set("failedRedirectUrl", fmt.Sprintf("%s/orders/%s", frontendURL, orderID.String()))
	params.Set("cancelRedirectUrl", fmt.Sprintf("%s/orders/%s", frontendURL, orderID.String()))

	// Sort and encode parameters alphabetically (excluding ApiSignature)
	encodedParams := params.Encode()

	// Generate signature on those parameters
	signature := generatePayrexxSignature(encodedParams, s.payrexxAPISecret)

	// Add ApiSignature to the POST parameters
	params.Set("ApiSignature", signature)
	finalBody := params.Encode()

	// Create POST request (signature sent in request body as POST parameter)
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(finalBody))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, "", fmt.Errorf("payrexx api returned status %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		Status string `json:"status"`
		Data   []struct {
			ID   int    `json:"id"`
			Link string `json:"link"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, "", err
	}

	if res.Status != "success" || len(res.Data) == 0 {
		return 0, "", fmt.Errorf("payrexx api returned failure status or empty data: %s", res.Status)
	}

	return res.Data[0].ID, res.Data[0].Link, nil
}

func generatePayrexxSignature(queryString string, apiSecret string) string {
	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(queryString))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) checkPayrexxGatewayStatus(gatewayID int) (string, error) {
	// API Endpoint
	apiURL := fmt.Sprintf("https://api.payrexx.com/v1.0/Gateway/%d/?instance=%s", gatewayID, s.payrexxInstance)

	// Signature needs to be computed on query string parameters excluding ApiSignature.
	// The only query parameter we pass is "instance".
	encodedParams := fmt.Sprintf("instance=%s", s.payrexxInstance)
	signature := generatePayrexxSignature(encodedParams, s.payrexxAPISecret)

	// Create URL with query params
	reqURL := fmt.Sprintf("%s&ApiSignature=%s", apiURL, url.QueryEscape(signature))

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("payrexx api returned status %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		Status string `json:"status"`
		Data   []struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if res.Status != "success" || len(res.Data) == 0 {
		return "", fmt.Errorf("payrexx api returned failure status or empty data: %s", res.Status)
	}

	return res.Data[0].Status, nil
}

