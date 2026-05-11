package services

import (
	"context"
	"errors"
	"fmt"

	"math"
	"strings"
	"time"

	"rest_api/internal/core/utils"
	"rest_api/internal/dto"
	"rest_api/internal/models"
	"rest_api/internal/repository"

	"github.com/google/uuid"
)

// --- STEP 1: THE INTERFACE ---
// This is what the routes and middleware "see"
type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (uuid.UUID, error)
	Login(ctx context.Context, req dto.LoginRequest) (models.User, string, string, bool, error)
	VerifyOTP(ctx context.Context, req dto.VerifyOTPRequest) (string, string, error)
	ResendOTP(ctx context.Context, email, otpType string) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error
	Logout(ctx context.Context, tokenString string) error
	GetMe(ctx context.Context, userID uuid.UUID) (*models.User, error)
	GetPaginatedUsers(ctx context.Context, page, limit int) ([]models.User, int, int, error)
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, block bool) error

	// THIS WAS THE MISSING METHOD CAUSING THE COMPILER ERROR
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
	UpdateUserEmail(ctx context.Context, userID uuid.UUID, newEmail string) error
}

// --- STEP 2: THE STRUCT ---
type authService struct {
	repo         repository.AuthRepository
	notifService NotificationService // This is correct
}

// Update the function signature to include notifService
func NewAuthService(repo repository.AuthRepository, notifService NotificationService) AuthService {
	return &authService{
		repo:         repo,
		notifService: notifService, // Now this works because it's passed in above
	}
}

// --- STEP 3: MISSING METHOD IMPLEMENTATION ---
func (s *authService) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	return s.repo.IsTokenBlacklisted(ctx, token)
}

// --- THE REST OF YOUR UPDATED METHODS ---

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (uuid.UUID, error) {
	cleanEmail, err := utils.CleanEmail(req.Email)
	if err != nil {
		return uuid.Nil, errors.New("invalid email")
	}

	if err := utils.ValidatePassword(req.Password); err != nil {
		return uuid.Nil, err
	}

	var finalRole models.Role
	if req.Role == "nanny" {
		finalRole = models.RoleNanny
	} else if req.Role == "employer" {
		finalRole = models.RoleEmployer
	} else {
		return uuid.Nil, errors.New("unauthorized: admin accounts cannot be created via registration")
	}

	hash, _ := utils.HashPassword(req.Password)

	newUser := models.User{
		ID:           uuid.New(),
		Email:        cleanEmail,
		PasswordHash: hash,
		Role:         finalRole,
		Status:       models.StatusPending,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.CreateUser(ctx, newUser); err != nil {
		return uuid.Nil, err
	}

	_ = s.TriggerOTP(ctx, newUser.ID, "email_verification")
	return newUser.ID, nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (models.User, string, string, bool, error) {
	cleanEmail, _ := utils.CleanEmail(req.Email)
	user, err := s.repo.GetByEmail(ctx, cleanEmail)
	if err != nil {
		return models.User{}, "", "", false, errors.New("Wrong email or password")
	}

	if user.Status == "pending" {
		return models.User{}, "", "", false, errors.New("please verify your email before logging in")
	}

	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		return models.User{}, "", "", false, errors.New("account locked")
	}

	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		s.handleFailedLogin(ctx, user)
		return models.User{}, "", "", false, errors.New("invalid credentials")
	}

	if user.Role == models.RoleAdmin {
		_ = s.TriggerOTP(ctx, user.ID, "admin_2fa")
		return user, "", "", true, nil
	}

	at, rt, _ := utils.GenerateTokens(user.ID, string(user.Role))
	_ = s.repo.UpdateFailedAttempts(ctx, user.ID, 0)

	return user, at, rt, false, nil
}

// ... Register, Login, etc. remain the same as they all call TriggerOTP ...

func (s *authService) TriggerOTP(ctx context.Context, userID uuid.UUID, otpType string) error {
	code := utils.GenerateOTP()
	fmt.Printf("\n[TERMINAL] Code for %s (%s): %s\n", userID, otpType, code)

	// 1. Save to Database for verification logic
	otp := models.OTP{
		UserID:    userID,
		Code:      code,
		Type:      otpType,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	if err := s.repo.SaveOTP(ctx, otp); err != nil {
		return err
	}

	// 2. QUEUE EMAIL NOTIFICATION
	// This will be picked up by the background worker
	_ = s.notifService.Send(ctx, models.NotificationRequest{
		UserID:    userID,
		EventType: otpType,
		Channels:  []string{"email"},
		Payload: models.NotificationPayload{
			Title: "Verification Code",
			Body:  fmt.Sprintf("Your %s code is: %s", otpType, code),
			Metadata: map[string]any{
				"otp": code,
			},
		},
	})

	return nil
}

func (s *authService) handleFailedLogin(ctx context.Context, user models.User) {
	count := user.FailedAttempts + 1
	_ = s.repo.UpdateFailedAttempts(ctx, user.ID, count)
	if count >= 5 {
		_ = s.repo.LockAccount(ctx, user.ID)
	}
}

func (s *authService) ForgotPassword(ctx context.Context, email string) error {
	cleanEmail, _ := utils.CleanEmail(email)
	user, err := s.repo.GetByEmail(ctx, cleanEmail)
	if err != nil {
		return nil
	}
	return s.TriggerOTP(ctx, user.ID, "password_reset")
}

func (s *authService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	cleanEmail, _ := utils.CleanEmail(req.Email)
	user, err := s.repo.GetByEmail(ctx, cleanEmail)
	if err != nil {
		return errors.New("invalid request")
	}

	if err := s.repo.VerifyAndExpireOTP(ctx, user.ID, req.Code, "password_reset"); err != nil {
		return err
	}

	if err := utils.ValidatePassword(req.NewPassword); err != nil {
		return err
	}

	newHash, _ := utils.HashPassword(req.NewPassword)
	return s.repo.UpdatePassword(ctx, cleanEmail, newHash)
}

func (s *authService) VerifyOTP(ctx context.Context, req dto.VerifyOTPRequest) (string, string, error) {
	cleanEmail, _ := utils.CleanEmail(req.Email)
	user, err := s.repo.GetByEmail(ctx, cleanEmail)
	if err != nil {
		return "", "", errors.New("user not found")
	}

	err = s.repo.VerifyAndExpireOTP(ctx, user.ID, req.Code, req.Type)
	if err != nil {
		return "", "", err
	}

	if req.Type == "email_verification" {
		_ = s.repo.UpdateUserStatus(ctx, user.ID, string(models.StatusActive))
	}

	return utils.GenerateTokens(user.ID, string(user.Role))
}

func (s *authService) ResendOTP(ctx context.Context, email, otpType string) error {
	cleanEmail, _ := utils.CleanEmail(email)
	user, err := s.repo.GetByEmail(ctx, cleanEmail)
	if err != nil {
		return errors.New("user not found")
	}
	return s.TriggerOTP(ctx, user.ID, otpType)
}

func (s *authService) Logout(ctx context.Context, tokenString string) error {
	claims, err := utils.ValidateToken(tokenString)
	if err != nil {
		return err
	}
	expirationTime := time.Unix(claims.ExpiresAt.Unix(), 0)
	return s.repo.BlacklistToken(ctx, tokenString, expirationTime)
}

func (s *authService) GetPaginatedUsers(ctx context.Context, page, limit int) ([]models.User, int, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	users, total, err := s.repo.GetPaginatedUsers(ctx, limit, offset)
	if err != nil {
		return nil, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	return users, total, totalPages, nil
}

func (s *authService) UpdateUserStatus(ctx context.Context, userID uuid.UUID, block bool) error {
	status := models.StatusActive
	if block {
		status = models.StatusBlocked
	}
	return s.repo.UpdateStatus(ctx, userID, status)
}

func (s *authService) GetMe(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

// users to updatethe email
func (s *authService) UpdateUserEmail(ctx context.Context, userID uuid.UUID, newEmail string) error {
	cleanEmail := strings.ToLower(strings.TrimSpace(newEmail))

	// 1. Call your existing GetByEmail
	// u is models.User (the struct), not a pointer
	_, err := s.repo.GetByEmail(ctx, cleanEmail)

	// 2. Logic: If there is NO error, it means a user WAS found.
	// This means the email is already taken.
	if err == nil {
		return errors.New("email address is already in use")
	}

	// 3. Logic: If the error is "No Rows", it means the email is FREE!
	// We check the error string since you're using pgx
	if err.Error() == "no rows in result set" || err.Error() == "pgx: no rows in result set" {
		// Proceed to update because the email is available
		err = s.repo.UpdateEmail(ctx, userID, cleanEmail)
		if err != nil {
			return fmt.Errorf("failed to update email: %w", err)
		}
		return nil
	}

	// 4. If it's any other error (like DB connection), return that error
	return fmt.Errorf("database error: %w", err)
}
