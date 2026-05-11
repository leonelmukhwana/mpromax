package services

import (
	"context"
	"errors"

	"time"

	"rest_api/internal/core/utils"
	"rest_api/internal/dto"
	"rest_api/internal/models"
	"rest_api/internal/repository"

	"github.com/google/uuid"
)

type NannyService interface {
	CreateNannyProfile(ctx context.Context, userID uuid.UUID, req dto.NannyProfileRequest) error
	UpdateNannyProfile(ctx context.Context, userID uuid.UUID, req dto.NannyUpdateProfileRequest) error
	GetMyProfile(ctx context.Context, userID uuid.UUID) (*dto.NannySelfViewResponse, error)
	AdminGetNanny(ctx context.Context, nannyID uuid.UUID) (*dto.NannyAdminViewResponse, error)
	AdminListNannies(ctx context.Context, filter models.NannySearchFilter) (*dto.PaginatedNannyResponse, error)
	DeleteNanny(ctx context.Context, nannyID uuid.UUID, actorID uuid.UUID, reason string) error
	RecoverNanny(ctx context.Context, nannyID uuid.UUID, actorID uuid.UUID) error
	GetNannyIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

type nannyService struct {
	repo repository.NannyRepository
}

func NewNannyService(repo repository.NannyRepository) NannyService {
	return &nannyService{repo: repo}
}

// 1. CREATE PROFILE (Sanitization + Age Check)
func (s *nannyService) CreateNannyProfile(ctx context.Context, userID uuid.UUID, req dto.NannyProfileRequest) error {
	// 1. Strict Name Validation (Rejects numbers like "John Doe33")
	cleanName, err := utils.SanitizeName(req.FullName)
	if err != nil {
		return err // This stops the process if the name has numbers
	}

	// 2. Phone Validation
	cleanPhone, err := utils.ValidatePhone(req.PhoneNumber)
	if err != nil {
		return err
	}

	// 3. Parse & Validate Age (Must be 18+)
	dob, err := time.Parse("2006-01-02", req.DOB)
	if err != nil {
		return errors.New("invalid date format, use YYYY-MM-DD")
	}

	tempNanny := &models.NannyProfile{DOB: dob}
	tempNanny.CalculateAge()
	if tempNanny.Age < 18 {
		return errors.New("nannies must be at least 18 years old")
	}

	// 4. Build Profile Struct
	profile := &models.NannyProfile{
		UserID:         userID,
		FullName:       cleanName,
		IDNumber:       req.IDNumber,
		PhoneNumber:    cleanPhone,
		DOB:            dob,
		HomeCounty:     req.HomeCounty,
		EducationLevel: req.EducationLevel,
	}

	// 5. Save to Repository (passing actorID as userID)
	return s.repo.CreateProfile(ctx, profile, userID)
}

func (s *nannyService) UpdateNannyProfile(ctx context.Context, userID uuid.UUID, req dto.NannyUpdateProfileRequest) error {
	// 1. Get existing profile
	existing, err := s.repo.GetProfileByID(ctx, userID)
	if err != nil {
		return errors.New("profile not found")
	}

	// 2. Apply updates with strict validation

	if req.PhoneNumber != nil {
		p, err := utils.ValidatePhone(*req.PhoneNumber)
		if err != nil {
			return err
		}
		existing.PhoneNumber = p
	}

	if req.HomeCounty != nil {
		existing.HomeCounty = *req.HomeCounty
	}

	if req.EducationLevel != nil {
		existing.EducationLevel = *req.EducationLevel
	}

	return s.repo.UpdateProfile(ctx, existing, userID)
}

// 3. ADMIN LIST (Pagination Calculation)
func (s *nannyService) AdminListNannies(ctx context.Context, f models.NannySearchFilter) (*dto.PaginatedNannyResponse, error) {
	// Set defaults if empty
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = 10
	}

	nannies, total, err := s.repo.ListNannies(ctx, f)
	if err != nil {
		return nil, err
	}

	var list []dto.NannyListResponse
	for _, n := range nannies {
		list = append(list, dto.NannyListResponse{
			UserID:         n.UserID,
			FullName:       n.FullName,
			PhoneNumber:    n.PhoneNumber,
			IDNumber:       n.IDNumber,
			Age:            n.Age,
			HomeCounty:     n.HomeCounty,
			EducationLevel: n.EducationLevel,
			IsVerified:     n.IsVerified,
			Status:         "Active",
		})
	}

	totalPages := (total + f.Limit - 1) / f.Limit

	return &dto.PaginatedNannyResponse{
		Data:       list,
		Total:      total,
		Page:       f.Page,
		TotalPages: totalPages,
	}, nil
}

// 4. DELETE & RECOVER (Orchestration)
func (s *nannyService) DeleteNanny(ctx context.Context, nannyID uuid.UUID, actorID uuid.UUID, reason string) error {
	return s.repo.SoftDeleteProfile(ctx, nannyID, actorID, reason)
}

func (s *nannyService) RecoverNanny(ctx context.Context, nannyID uuid.UUID, actorID uuid.UUID) error {
	return s.repo.RecoverProfile(ctx, nannyID, actorID)
}

// 5. VIEW PROFILE (Mapping to DTOs)
func (s *nannyService) GetMyProfile(ctx context.Context, userID uuid.UUID) (*dto.NannySelfViewResponse, error) {
	p, err := s.repo.GetProfileByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.NannySelfViewResponse{
		FullName:       p.FullName,
		PhoneNumber:    p.PhoneNumber,
		IDNumber:       p.IDNumber,
		Age:            p.Age,
		HomeCounty:     p.HomeCounty,
		EducationLevel: p.EducationLevel,
		IsVerified:     p.IsVerified,
	}, nil
}

// 6. ADMIN GET SINGLE (Decrypted for manual verification)
func (s *nannyService) AdminGetNanny(ctx context.Context, nannyID uuid.UUID) (*dto.NannyAdminViewResponse, error) {
	// Fetch from repo (The repo already handles decryption in GetProfileByID)
	p, err := s.repo.GetProfileByID(ctx, nannyID)
	if err != nil {
		return nil, errors.New("nanny profile not found")
	}

	// Map to Admin-specific DTO
	return &dto.NannyAdminViewResponse{
		UserID:         p.UserID,
		FullName:       p.FullName,
		IDNumber:       p.IDNumber,    // Decrypted ID
		PhoneNumber:    p.PhoneNumber, // Decrypted Phone
		Age:            p.Age,
		HomeCounty:     p.HomeCounty,
		EducationLevel: p.EducationLevel,
		FCMToken:       p.FCMToken,
		IsVerified:     p.IsVerified,
		CreatedAt:      p.CreatedAt,
	}, nil
}

func (s *nannyService) GetNannyIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	// This calls the repository method we created earlier
	return s.repo.GetNannyIDByUserID(ctx, userID)
}
