package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/M306/backend/internal/db/sqlc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type MockRoundTripper func(req *http.Request) *http.Response

func (f MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestPayrexxWebhook(t *testing.T) {
	server := NewServer(testQueries, testDB, &mockStorage{}, &MockEmailService{}, "secret", "test-instance", "api-secret-key")
	router := server.Routes()

	// 1. Create mock tenant
	tenant, err := testQueries.CreateTenant(context.Background(), db.CreateTenantParams{
		Name: "Payment Webhook Test Farm",
		Slug: "payment-webhook-test-farm",
	})
	require.NoError(t, err)

	// 2. Create mock user
	user, err := testQueries.CreateUser(context.Background(), db.CreateUserParams{
		TenantID:     uuid.NullUUID{UUID: tenant.ID, Valid: true},
		Email:        "user-webhook@paymenttest.com",
		PasswordHash: "somehash",
		Role:         "customer",
	})
	require.NoError(t, err)

	// 3. Create mock order
	order, err := testQueries.CreateOrder(context.Background(), db.CreateOrderParams{
		TenantID:         tenant.ID,
		UserID:           user.ID,
		Status:           "pending_payment",
		TotalAmount:      "250.00",
		PaymentMethod:    "online",
		PayrexxGatewayID: sql.NullInt32{Int32: 12345, Valid: true},
		PaymentStatus:    "unpaid",
	})
	require.NoError(t, err)

	// 4. Mock the webhook payload
	payload := []byte(fmt.Sprintf("transaction[status]=confirmed&transaction[referenceId]=%s", order.ID.String()))
	mac := hmac.New(sha256.New, []byte("api-secret-key"))
	mac.Write(payload)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// 5. Make the request
	req, err := http.NewRequest("POST", "/api/v1/payments/webhook", bytes.NewBuffer(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Webhook-Signature", signature)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	// 6. Verify database order is updated
	updatedOrder, err := testQueries.GetOrder(context.Background(), order.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", updatedOrder.Status)
	require.Equal(t, "paid", updatedOrder.PaymentStatus)
}

func TestGetOrderDetailsWithRealtimePayrexxCheck(t *testing.T) {
	server := NewServer(testQueries, testDB, &mockStorage{}, &MockEmailService{}, "secret", "test-instance", "api-secret-key")
	router := server.Routes()

	// 1. Create mock tenant
	tenant, err := testQueries.CreateTenant(context.Background(), db.CreateTenantParams{
		Name: "Payment Details Test Farm",
		Slug: "payment-details-test-farm",
	})
	require.NoError(t, err)

	// 2. Create mock user
	user, err := testQueries.CreateUser(context.Background(), db.CreateUserParams{
		TenantID:     uuid.NullUUID{UUID: tenant.ID, Valid: true},
		Email:        "user-details@paymenttest.com",
		PasswordHash: "somehash",
		Role:         "customer",
	})
	require.NoError(t, err)

	// 3. Create mock order
	order, err := testQueries.CreateOrder(context.Background(), db.CreateOrderParams{
		TenantID:         tenant.ID,
		UserID:           user.ID,
		Status:           "pending_payment",
		TotalAmount:      "320.00",
		PaymentMethod:    "online",
		PayrexxGatewayID: sql.NullInt32{Int32: 67890, Valid: true},
		PaymentStatus:    "unpaid",
	})
	require.NoError(t, err)

	// 4. Mock the http.DefaultTransport RoundTripper to intercept outgoing calls to Payrexx API
	oldTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = oldTransport }()

	http.DefaultTransport = MockRoundTripper(func(req *http.Request) *http.Response {
		require.Contains(t, req.URL.String(), "api.payrexx.com/v1.0/Gateway/67890")
		require.Equal(t, "GET", req.Method)

		// Verify API Signature query parameters
		q := req.URL.Query()
		require.Equal(t, "test-instance", q.Get("instance"))
		require.NotEmpty(t, q.Get("ApiSignature"))

		// Mock the Payrexx gateway status response
		respBody := `{
			"status": "success",
			"data": [
				{
					"id": 67890,
					"status": "confirmed"
				}
			]
		}`

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(respBody)),
		}
	})

	// 5. Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte("secret"))
	require.NoError(t, err)

	// 6. Make request to GET /api/v1/orders/{id}
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/v1/orders/%s", order.ID.String()), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	// 7. Verify response contains updated order status
	var res struct {
		Order db.Order `json:"order"`
	}
	err = json.NewDecoder(rr.Body).Decode(&res)
	require.NoError(t, err)
	require.Equal(t, "completed", res.Order.Status)
	require.Equal(t, "paid", res.Order.PaymentStatus)

	// 8. Verify order is updated in DB
	dbOrder, err := testQueries.GetOrder(context.Background(), order.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", dbOrder.Status)
	require.Equal(t, "paid", dbOrder.PaymentStatus)
}

func TestCreateOrderWithShippingAddressOverride(t *testing.T) {
	server := NewServer(testQueries, testDB, &mockStorage{}, &MockEmailService{}, "secret", "test-instance", "api-secret-key")
	router := server.Routes()

	// 1. Create mock tenant (ensuring allows_cash_payment is true)
	tenant, err := testQueries.CreateTenant(context.Background(), db.CreateTenantParams{
		Name: "Shipping Override Farm",
		Slug: "shipping-override-farm",
	})
	require.NoError(t, err)
	_, err = testQueries.UpdateTenantPaymentSettings(context.Background(), db.UpdateTenantPaymentSettingsParams{
		ID:                  tenant.ID,
		AllowsOnlinePayment: true,
		AllowsCashPayment:   true,
	})
	require.NoError(t, err)

	// 2. Create mock user (with address)
	user, err := testQueries.CreateUser(context.Background(), db.CreateUserParams{
		TenantID:     uuid.NullUUID{UUID: tenant.ID, Valid: true},
		Email:        "user-shipping@paymenttest.com",
		PasswordHash: "somehash",
		Role:         "customer",
	})
	require.NoError(t, err)
	_, err = testQueries.UpdateUserProfile(context.Background(), db.UpdateUserProfileParams{
		ID:       user.ID,
		FullName: sql.NullString{String: "Default Name", Valid: true},
		Street:   sql.NullString{String: "Default Street 1", Valid: true},
		ZipCode:  sql.NullString{String: "1000", Valid: true},
		City:     sql.NullString{String: "Default City", Valid: true},
	})
	require.NoError(t, err)

	// 3. Create mock product
	product, err := testQueries.CreateProduct(context.Background(), db.CreateProductParams{
		TenantID:    tenant.ID,
		Name:        "Apple",
		Description: sql.NullString{String: "Fresh", Valid: true},
		Price:       "2.50",
		Stock:       100,
		ImageUrl:    sql.NullString{String: "", Valid: true},
		Category:    sql.NullString{String: "Fruit", Valid: true},
	})
	require.NoError(t, err)

	// 4. Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte("secret"))
	require.NoError(t, err)

	// 5. Make request to POST /api/v1/orders with custom address fields
	reqBody, err := json.Marshal(map[string]interface{}{
		"payment_method": "cash",
		"items": []map[string]interface{}{
			{
				"product_id": product.ID.String(),
				"quantity":   2,
			},
		},
		"street":    "Override Street 999",
		"zip_code":   "8888",
		"city":       "Override Town",
		"full_name":  "Override Full Name",
	})
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-Slug", "shipping-override-farm")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var res struct {
		Order db.Order `json:"order"`
	}
	err = json.NewDecoder(rr.Body).Decode(&res)
	require.NoError(t, err)

	// 6. Verify that the order record has the custom shipping address in the DB
	dbOrder, err := testQueries.GetOrder(context.Background(), res.Order.ID)
	require.NoError(t, err)
	require.Equal(t, "Override Street 999", dbOrder.ShippingStreet.String)
	require.Equal(t, "8888", dbOrder.ShippingZipCode.String)
	require.Equal(t, "Override Town", dbOrder.ShippingCity.String)
	require.Equal(t, "Override Full Name", dbOrder.ShippingFullName.String)
}

