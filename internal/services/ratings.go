package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"rest_api/internal/core/utils"
	"rest_api/internal/dto"
	"rest_api/internal/models"
	"rest_api/internal/repository"

	"github.com/google/uuid"
)

// RatingService defines the business logic contract.
type RatingService interface {
	SubmitMonthlyRating(ctx context.Context, req dto.CreateMonthlyRatingRequest) error
	GetAdminRatingDashboard(ctx context.Context, filter dto.RatingFilterParams) (dto.AdminRatingListResponse, error)
	GetNannyRatingHistory(ctx context.Context, nannyID uuid.UUID, filter dto.RatingFilterParams) (dto.NannyRatingListResponse, error)
	// REMOVED: GetNannyIDByUserID (This should be in NannyService)
}

type ratingService struct {
	repo repository.RatingRepository
}

func NewRatingService(repo repository.RatingRepository) RatingService {
	return &ratingService{
		repo: repo,
	}
}

// 1. SubmitMonthlyRating: Orchestrates validation and persistence.
func (s *ratingService) SubmitMonthlyRating(ctx context.Context, req dto.CreateMonthlyRatingRequest) error {
	now := time.Now()
	if req.Year > now.Year() || (req.Year == now.Year() && req.Month > int(now.Month())) {
		return fmt.Errorf("validation_error: cannot submit a rating for a future period")
	}

	rating := &models.MonthlyRating{
		ID:           uuid.New(),
		AssignmentID: req.AssignmentID,
		EmployerID:   req.EmployerID,
		NannyID:      req.NannyID,
		RatingValue:  req.RatingValue,
		ReviewText:   req.ReviewText,
		RatingMonth:  req.Month,
		RatingYear:   req.Year,
	}

	if err := s.repo.CreateMonthlyRating(ctx, rating); err != nil {
		return fmt.Errorf("service_error: %w", err)
	}

	return nil
}

// 2. GetAdminRatingDashboard: Full decryption and pagination logic.
func (s *ratingService) GetAdminRatingDashboard(ctx context.Context, filter dto.RatingFilterParams) (dto.AdminRatingListResponse, error) {
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	items, total, err := s.repo.GetAdminRatingDashboard(ctx, filter)
	if err != nil {
		return dto.AdminRatingListResponse{}, err
	}

	if items == nil {
		items = []dto.AdminRatingItem{}
	}

	// Decrypt sensitive info
	for i := range items {
		if items[i].NannyPhoneNumber != "" && len(items[i].NannyPhoneNumber) > 15 {
			decryptedPhone, err := utils.Decrypt(items[i].NannyPhoneNumber)
			if err != nil {
				log.Printf("Decryption failed for Admin Dashboard item %d: %v", i, err)
				continue
			}
			items[i].NannyPhoneNumber = decryptedPhone
		}
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(filter.Limit) - 1) / int64(filter.Limit))
	}

	return dto.AdminRatingListResponse{
		Data: items,
		Meta: dto.PaginationMeta{
			TotalRecords: total,
			TotalPages:   totalPages,
			CurrentPage:  filter.Page,
			PageSize:     filter.Limit,
		},
	}, nil
}

// 3. GetNannyRatingHistory: Provides a filtered view for the Nanny.
func (s *ratingService) GetNannyRatingHistory(ctx context.Context, nannyID uuid.UUID, filter dto.RatingFilterParams) (dto.NannyRatingListResponse, error) {
	// Set defaults
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	data, total, err := s.repo.GetNannyRatingsPaginated(ctx, nannyID, filter)
	if err != nil {
		return dto.NannyRatingListResponse{}, err
	}

	// Initialize empty slice for JSON safety
	if data == nil {
		data = []dto.NannyRatingItem{}
	}

	// Decryption for Review Texts/Employer Names if they were encrypted
	// (Apply the same logic used in Admin Dashboard if necessary)

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(filter.Limit) - 1) / int64(filter.Limit))
	}

	return dto.NannyRatingListResponse{
		Data: data,
		Meta: dto.PaginationMeta{
			TotalRecords: total,
			TotalPages:   totalPages,
			CurrentPage:  filter.Page,
			PageSize:     filter.Limit,
		},
	}, nil
}
