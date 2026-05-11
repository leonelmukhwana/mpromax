package services

import (
	"context"
	"errors"
	"rest_api/internal/dto"
	"rest_api/internal/models"
	"rest_api/internal/repository"
	"time"

	"github.com/google/uuid"
)

type BookingService interface {
	CreateBooking(ctx context.Context, nannyID uuid.UUID, slot time.Time, idempotencyKey string) (*models.Booking, error)
	GetMyBookings(ctx context.Context, nannyID uuid.UUID, page, limit int) ([]models.Booking, int64, error)
	GetAdminList(ctx context.Context, filter models.AdminBookingFilter) ([]dto.AdminBookingResponse, int64, error)
}

type bookingService struct {
	repo *repository.BookingRepository
}

func NewBookingService(repo *repository.BookingRepository) BookingService {
	return &bookingService{repo: repo}
}

func (s *bookingService) CreateBooking(ctx context.Context, nannyID uuid.UUID, slot time.Time, idempotencyKey string) (*models.Booking, error) {
	// 1. Time Window Validation (8:00 AM - 5:00 PM)
	hour := slot.Hour()
	if hour < 8 || hour >= 17 {
		return nil, errors.New("interviews must be scheduled between 08:00 and 17:00")
	}

	// 2. Enforce 30-Minute Intervals
	if slot.Minute() != 0 && slot.Minute() != 30 {
		return nil, errors.New("invalid time slot: please choose a time on the hour or half-hour (e.g., 09:00 or 09:30)")
	}

	// 3. Sanitize the Timestamp
	sanitizedSlot := time.Date(
		slot.Year(), slot.Month(), slot.Day(),
		slot.Hour(), slot.Minute(), 0, 0, slot.Location(),
	)

	// 4. Check for Past Dates
	if sanitizedSlot.Before(time.Now()) {
		return nil, errors.New("cannot book a slot in the past")
	}

	// 5. Prepare Model
	booking := &models.Booking{
		NannyID:        nannyID,
		BookingSlot:    sanitizedSlot,
		IdempotencyKey: idempotencyKey,
	}

	// 6. Call Repository
	if err := s.repo.CreateBooking(ctx, booking); err != nil {
		return nil, err
	}

	return booking, nil
}

func (s *bookingService) GetAdminList(ctx context.Context, filter models.AdminBookingFilter) ([]dto.AdminBookingResponse, int64, error) {
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	return s.repo.GetAdminBookings(ctx, filter)
}

func (s *bookingService) GetMyBookings(ctx context.Context, nannyID uuid.UUID, page, limit int) ([]models.Booking, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	return s.repo.GetNannyBookings(ctx, nannyID, limit, offset)
}
