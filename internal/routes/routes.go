package routes

import (
	"rest_api/internal/core/middleware"
	"rest_api/internal/handlers"
	"rest_api/internal/services"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	r *gin.Engine,
	authHandler *handlers.AuthHandler,
	nannyHandler *handlers.NannyHandler,
	clientHandler *handlers.ClientHandler,
	verificationHandler *handlers.VerificationHandler,
	jobHandler *handlers.JobHandler,
	bookingHandler *handlers.BookingHandler,
	assignmentHandler *handlers.AssignmentHandler,
	contractHandler *handlers.ContractHandler,
	ratingHandler *handlers.RatingHandler,
	incidentHandler *handlers.IncidentHandler,
	paymentHandler *handlers.PaymentHandler,
	notifHandler *handlers.NotificationHandler, // Added Notification Dependency
	authService services.AuthService,
) {

	// --- GLOBAL MIDDLEWARE ---
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.RateLimitMiddleware())

	// --- PUBLIC AUTH ROUTES ---
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/verify", authHandler.VerifyOTP)
		auth.POST("/resend", authHandler.ResendOTP)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)
		auth.POST("/webhooks/mpesa", paymentHandler.MpesaWebhook)
	}

	// --- PROTECTED BASE ROUTES (Shared) ---
	commonAuth := r.Group("/api/v1")
	commonAuth.Use(middleware.AuthMiddleware(authService))
	{
		commonAuth.POST("/auth/logout", authHandler.Logout)
		commonAuth.POST("/incidents", incidentHandler.SubmitReport)
		commonAuth.GET("/my-reports", incidentHandler.GetMyReports)
		commonAuth.GET("/payments", paymentHandler.GetUserPayments)
		commonAuth.GET("/notifications", notifHandler.GetMyHistory)
		commonAuth.PATCH("/profile/email", authHandler.UpdateEmail)
	}

	// --- NANNY ONLY ROUTES ---
	nanny := r.Group("/api/v1/profiles/nanny")
	nanny.Use(middleware.AuthMiddleware(authService), middleware.NannyOnly())
	{
		nanny.POST("/", nannyHandler.Create)
		nanny.GET("/me", nannyHandler.GetMe)
		nanny.PATCH("/me", nannyHandler.UpdateMe)
		nanny.POST("/verify", verificationHandler.Upload)
		nanny.GET("/verify/status", verificationHandler.GetStatus)
		nanny.POST("/bookings", bookingHandler.Create)
		nanny.GET("/bookings", bookingHandler.GetMyBookings)
		nanny.GET("/assignments/:id", assignmentHandler.GetForNanny)
		nanny.GET("/assignments/:id/contract", contractHandler.DownloadContract)
		nanny.GET("/ratings", ratingHandler.GetNannyHistory)
		nanny.GET("/earnings", paymentHandler.GetUserPayments)
	}

	// --- EMPLOYER (CLIENT) ROUTES ---
	employer := r.Group("/api/v1/profiles/client")
	employer.Use(middleware.AuthMiddleware(authService), middleware.EmployerOnly())
	{
		employer.POST("/", clientHandler.CreateProfile)
		employer.GET("/me", clientHandler.GetMyProfile)
		employer.PUT("/me", clientHandler.UpdateProfile)
		employer.POST("/jobs", jobHandler.Create)
		employer.PATCH("/jobs/:id", jobHandler.Update)
		employer.GET("/jobs/:id", jobHandler.GetJob)
		employer.GET("/assignments/:id/contract", contractHandler.DownloadContract)
		employer.POST("/ratings", ratingHandler.SubmitRating)

		// Employer Payment Actions
		employer.GET("/billing", paymentHandler.GetUserPayments)
		employer.POST("/pay-stk", paymentHandler.InitiateSTKPush)
	}

	// --- ADMIN ROUTES ---
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AuthMiddleware(authService), middleware.AdminOnly())
	{
		admin.GET("/nannies", nannyHandler.AdminList)
		admin.GET("/clients", clientHandler.AdminList)
		admin.GET("/jobs", jobHandler.AdminList)
		admin.GET("/users", authHandler.ListUsers)
		admin.GET("/verifications", verificationHandler.AdminGetVerifications)
		admin.GET("/bookings", bookingHandler.List)
		admin.POST("/assignments", assignmentHandler.Create)
		admin.GET("/assignments", assignmentHandler.GetAll)
		admin.GET("/assignments/:id/contract", contractHandler.DownloadContract)
		admin.POST("/assignments/:id/regenerate", contractHandler.AdminForceGenerate)
		admin.GET("/ratings", ratingHandler.GetAdminDashboard)
		admin.GET("/incidents", incidentHandler.AdminListReports)
		admin.PATCH("/incidents/:id/status", incidentHandler.UpdateIncidentStatus)
		admin.GET("/payments", paymentHandler.GetUserPayments)
		admin.GET("/payments/dashboard", ratingHandler.GetAdminDashboard)
		admin.PATCH("/users/:id/status", authHandler.ToggleBlock)

		// ADMIN NOTIFICATION TOOLS
		// Used for manual system alerts and testing providers
		admin.POST("/notifications/dispatch", notifHandler.SendManualNotification)
		admin.GET("/notifications/outbox", notifHandler.GetOutboxStatus)
	}
}
