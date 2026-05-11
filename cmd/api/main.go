package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth_gin"
	"rest_api/internal/core/database"
	"rest_api/internal/core/middleware"
	"rest_api/internal/core/utils"
	"rest_api/internal/handlers"
	"rest_api/internal/pkg/notifications"
	"rest_api/internal/pkg/worker"
	"rest_api/internal/repository"
	"rest_api/internal/routes"
	"rest_api/internal/services"
)

var NairobiLoc *time.Location

func init() {
    var err error
    NairobiLoc, err = time.LoadLocation("Africa/Nairobi")
    if err != nil {
        log.Printf("Warning: Could not load Nairobi location: %v", err)
        NairobiLoc = time.FixedZone("EAT", 3*3600) // Fallback to UTC+3
    }
}

type NilLogger struct{}

func (l *NilLogger) Log(ctx context.Context, tx pgx.Tx, action, actorID, entityID string) error {
	return nil
}

func main() {
	// 1. Load Environment Variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// 2. Setup Database Variables with proper fallbacks for hosting
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	sslMode := os.Getenv("DB_SSL_MODE")
	if sslMode == "" {
		sslMode = "disable" // Use "require" on platforms like Render
	}

	// 3. Build the Connection String for Migrations and pgx
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPass, dbHost, dbPort, dbName, sslMode)

	// 4. Run migrations BEFORE connecting the pool
	// This ensures the schema is ready before services start up
	database.RunMigrations(dbURL)

	// 5. Initialize Database Connection Pool
	dbPool, err := database.ConnectDB()
	if err != nil {
		log.Fatalf("CRITICAL: Could not connect to DB: %v", err)
	}
	defer dbPool.Close()

	// 6. External Integrations
	cld, _ := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)

	// --- 7. UTILITIES & PROVIDERS ---
	pdfUtil := utils.NewPDFGenerator()
	cloudUtil := utils.NewCloudinaryUtil(cld)

	emailProv := notifications.NewEmailProvider()
	fcmProv := &notifications.FCMProvider{}
	webProv := &notifications.WebProvider{}

	// --- 8. REPOSITORIES ---
	authRepo := repository.NewAuthRepository(dbPool)
	nannyRepo := repository.NewNannyRepository(dbPool)
	clientRepo := repository.NewClientRepository(dbPool)
	verifyRepo := repository.NewVerificationRepository(dbPool)
	jobRepo := repository.NewJobRepository(dbPool)
	bookingRepo := repository.NewBookingRepository(dbPool)
	assignmentRepo := repository.NewAssignmentRepository(dbPool)
	contractRepo := repository.NewContractRepository(dbPool)
	ratingRepo := repository.NewRatingRepository(dbPool, dbPool)
	incidentRepo := repository.NewIncidentRepository(dbPool)
	paymentRepo := repository.NewPaymentRepository(dbPool)
	notifRepo := repository.NewNotificationRepository(dbPool)

	// --- 9. SERVICES ---
	notifService := services.NewNotificationService(notifRepo, authRepo, nannyRepo, clientRepo)
	authService := services.NewAuthService(authRepo, notifService)
	nannyService := services.NewNannyService(nannyRepo)
	auditLogger := &NilLogger{}

	clientService := services.NewClientService(dbPool, clientRepo, authRepo, auditLogger)
	verifyService := services.NewVerificationService(verifyRepo, cld)
	jobService := services.NewJobService(dbPool, jobRepo)
	bookingService := services.NewBookingService(bookingRepo)
	contractService := services.NewContractService(contractRepo, pdfUtil, cloudUtil)
	assignmentService := services.NewAssignmentService(assignmentRepo, contractService, notifService)
	ratingService := services.NewRatingService(ratingRepo)
	incidentService := services.NewIncidentService(incidentRepo, nannyRepo)

	mpesaSvc := &services.MpesaService{
		Key:         os.Getenv("MPESA_KEY"),
		Secret:      os.Getenv("MPESA_SECRET"),
		Passkey:     os.Getenv("MPESA_PASSKEY"),
		ShortCode:   os.Getenv("MPESA_SHORTCODE"),
		CallbackURL: os.Getenv("MPESA_CALLBACK_URL"),
	}

	if mpesaSvc == nil {
		log.Fatal("Failed to initialize MpesaService")
	}
	paymentService := services.NewPaymentService(paymentRepo, notifService, mpesaSvc)

	// --- 10. BACKGROUND WORKERS ---
	notifWorker := worker.NewNotificationWorker(notifRepo, emailProv, fcmProv, webProv)
	go notifWorker.Start(context.Background())

	// --- 11. HANDLERS ---
	authHandler := handlers.NewAuthHandler(authService)
	nannyHandler := handlers.NewNannyHandler(nannyService)
	clientHandler := handlers.NewClientHandler(clientService)
	verifyHandler := handlers.NewVerificationHandler(verifyService)
	jobHandler := handlers.NewJobHandler(jobService)
	bookingHandler := handlers.NewBookingHandler(bookingService)
	assignmentHandler := handlers.NewAssignmentHandler(assignmentService)
	contractHandler := handlers.NewContractHandler(contractService)
	ratingHandler := handlers.NewRatingHandler(ratingService, nannyService)
	incidentHandler := handlers.NewIncidentHandler(incidentService)
	paymentHandler := handlers.NewPaymentHandler(paymentService)
	notifHandler := handlers.NewNotificationHandler(notifService)

	// --- 12. ROUTES & SERVER ---
	r := gin.Default()
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.RateLimitMiddleware())

	// Limit to 1 request per second (stops spamming)
	limiter := tollbooth.NewLimiter(1, nil)
	r.Use(tollbooth_gin.LimitHandler(limiter))

	// CORS Configuration
		// Get origins from environment, or use a safe local default
		originsEnv := os.Getenv("ALLOWED_ORIGINS")

		// 2. Set a default if the environment variable is missing
		if originsEnv == "" {
			originsEnv = "http://localhost:3000,http://localhost:5000"
		}

		// 3. Split the comma-separated string into a slice (list)
		allowedOrigins := strings.Split(originsEnv, ",")

		r.Use(cors.New(cors.Config{
			AllowOrigins:     allowedOrigins, // Now it uses the list from .env!
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))

	// Health Check Route for Render/Monitoring
		r.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "up",
				"message": "MProMax API is running",
				"time":    time.Now().Format(time.RFC3339),
			})
		})

	routes.SetupRoutes(
		r,
		authHandler,
		nannyHandler,
		clientHandler,
		verifyHandler,
		jobHandler,
		bookingHandler,
		assignmentHandler,
		contractHandler,
		ratingHandler,
		incidentHandler,
		paymentHandler,
		notifHandler,
		authService,
	)

	// Get Port from environment for hosting
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("🚀 MProMax API starting on port %s", port)
	// Listen on 0.0.0.0 to ensure the host can route traffic to the app
	if err := r.Run("0.0.0.0:" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
