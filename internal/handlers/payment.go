package handlers

import (
	"fmt"
	"net/http"
	"rest_api/internal/dto"
	"rest_api/internal/repository"
	"rest_api/internal/services"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type UserStats struct {
	Attempts    int
	LockedUntil time.Time
}

type PaymentHandler struct {
	svc          *services.PaymentService
	repo         *repository.PaymentRepository
	mpesaService *services.MpesaService
	mu           sync.Mutex
	voters       map[string]*UserStats
}

// Include the repository (r) and mpesa service (m) in the arguments
func NewPaymentHandler(svc *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		svc: svc,
	}
}

// InitiateSTKPush: Triggered when the EMPLOYER clicks "Pay" in your app
func (h *PaymentHandler) InitiateSTKPush(c *gin.Context) {

	// 1. Get User ID from Context
	val, exists := c.Get("user_id")
	if !exists {
		
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: No user session found"})
		return
	}
	employerID := fmt.Sprintf("%v", val)

	// --- RATE LIMIT LOGIC START ---
	h.mu.Lock()
	if h.voters == nil {
		h.voters = make(map[string]*UserStats)
	}

	stats, found := h.voters[employerID]
	if !found {
		stats = &UserStats{Attempts: 0}
		h.voters[employerID] = stats
	}

	now := time.Now()

	// 1. If currently locked, stay locked
	if now.Before(stats.LockedUntil) {
		remaining := time.Until(stats.LockedUntil).Round(time.Second)
		h.mu.Unlock()
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": fmt.Sprintf("Too many requests. Please wait %v.", remaining),
		})
		return
	}

	// 2. If the lock time has passed, reset the counter for a fresh start
	if now.After(stats.LockedUntil) && !stats.LockedUntil.IsZero() {
		stats.Attempts = 0
		stats.LockedUntil = time.Time{} // Clear the lock
	}

	// 3. Increment AFTER checking the lock
	stats.Attempts++

	// 4. On the 3rd attempt, set the lock for NEXT time
	if stats.Attempts >= 3 {
		stats.LockedUntil = now.Add(5 * time.Minute)
		fmt.Printf("User %s reached 3 attempts. Locking for 5 mins.\n", employerID)
	}
	h.mu.Unlock()
	// --- RATE LIMIT LOGIC END ---

	// 2. Bind JSON Request
	var req dto.STKPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {

		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: phone_number and job_ref required"})
		return
	}

	// 3. Nil-Pointer Protection for Service
	if h.svc == nil {
		fmt.Println("PaymentHandler.svc is nil!")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment service not initialized"})
		return
	}

	// 4. Call Service to Trigger STK
	fmt.Printf("Calling service for JobRef: %s, Phone: %s\n", req.JobRef, req.PhoneNumber)
	err := h.svc.InitiateSTK(c.Request.Context(), employerID, req)
	if err != nil {
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "M-Pesa STK Push failed",
			"details": err.Error(),
		})
		return
	}

	// 5. Success!
	c.JSON(http.StatusOK, gin.H{
		"message": "STK Push initiated successfully. Please check your phone for the PIN prompt.",
		"job_ref": req.JobRef,
	})
}

// MpesaWebhook: Receives data when EMPLOYER pays (STK or Manual Toolkit)
func (h *PaymentHandler) MpesaWebhook(c *gin.Context) {
	var input dto.MpesaWebhookInput
	if err := c.ShouldBindJSON(&input); err != nil {
		// Log the raw error here to see why binding failed
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook data"})
		return
	}

	// Process the 10/90 split automatically
	err := h.svc.ProcessMpesaPayment(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process payment"})
		return
	}

	// Safaricom requires a success response
	c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Success"})
}

// GetUserPayments: Routes to Nanny, Admin, or Employer tables
func (h *PaymentHandler) GetUserPayments(c *gin.Context) {
	var req dto.PaymentPaginationRequest

	// 1. Bind query params (?page=1&limit=10&search=xyz)
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "check your filters"})
		return
	}

	// Default pagination values
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	// 2. Extract UserID Safely
	rawUserID, _ := c.Get("user_id")
	var userID string
	switch v := rawUserID.(type) {
	case string:
		userID = v
	case interface{ String() string }:
		userID = v.String()
	default:
		userID = fmt.Sprintf("%v", v)
	}

	// 3. Extract Role Safely (Checking multiple common keys)
	var roleStr string
	if r, ok := c.Get("role"); ok && r != nil {
		roleStr = fmt.Sprintf("%v", r)
	} else if r, ok := c.Get("user_role"); ok && r != nil {
		roleStr = fmt.Sprintf("%v", r)
	}

	// DEBUG: Verify exactly what is being sent to the service
	fmt.Printf("Handler passing to Service -> UserID: %s | Role: %s\n", userID, roleStr)

	// 4. Call the Service 'Unified Fetcher'
	data, total, err := h.svc.GetPaymentsForUser(
		c.Request.Context(),
		userID,
		roleStr,
		req,
	)

	if err != nil {
		// If the service still says "unauthorized", we print it to terminal to see why
		fmt.Printf("Service Layer Error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 5. Success Response
	if data == nil {
		data = []dto.PaymentResponse{}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   data,
		"meta": gin.H{
			"total": total,
			"page":  req.Page,
			"limit": req.Limit,
		},
	})
}

// mpesa callback
func (h *PaymentHandler) MpesaCallback(c *gin.Context) {
	var callbackData map[string]interface{}
	if err := c.ShouldBindJSON(&callbackData); err != nil {
		return
	}

	// Navigate the Safaricom JSON structure
	body := callbackData["Body"].(map[string]interface{})
	stkCallback := body["stkCallback"].(map[string]interface{})
	resultCode := stkCallback["ResultCode"].(float64)

	if resultCode == 0 {
		// Payment Successful!
		meta := stkCallback["CallbackMetadata"].(map[string]interface{})
		items := meta["Item"].([]interface{})

		var receipt, phone string
		var amount float64

		for _, item := range items {
			val := item.(map[string]interface{})
			switch val["Name"] {
			case "MpesaReceiptNumber":
				receipt = val["Value"].(string)
			case "Amount":
				amount = val["Value"].(float64)
			case "PhoneNumber":
				phone = fmt.Sprintf("%v", val["Value"])
			}
		}

		// LOGIC: Use the JobRef (AccountReference) to find the assignment
		// and save to your payments table using your Repository
		fmt.Printf("Success! Receipt: %s, Amount: %f, Phone: %s\n", receipt, amount, phone)

		// h.repo.CreatePaymentAfterMpesa(...)
	}

	c.JSON(200, gin.H{"ResultCode": 0, "ResultDesc": "Success"})
}

func (h *PaymentHandler) MpesaValidation(c *gin.Context) {
	var req struct {
		BillRef     string  `json:"BillRefNumber"`
		TransAmount float64 `json:"TransAmount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	// FIX: Receive 3 values (exists, expectedAmount, err)
	// and pass the context (c.Request.Context())
	exists, expectedAmount, err := h.repo.VerifyJobRef(c.Request.Context(), req.BillRef)

	// Always check if the database query itself failed
	if err != nil {
		c.JSON(200, gin.H{
			"ResultCode": "C2B00016", // Generic Error code for Safaricom
			"ResultDesc": "Internal System Error",
		})
		return
	}

	// Now validate the business logic
	if !exists || req.TransAmount < expectedAmount {
		c.JSON(200, gin.H{
			"ResultCode": "C2B00012",
			"ResultDesc": "Rejected: Invalid Job Reference or Amount",
		})
		return
	}

	// Success!
	c.JSON(200, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
}
