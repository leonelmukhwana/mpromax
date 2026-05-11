package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rest_api/internal/core/utils"
	"rest_api/internal/dto"
	"rest_api/internal/models"
	"rest_api/internal/repository"
)

type ClientRepository interface {
	Create(ctx context.Context, tx pgx.Tx, p *models.ClientProfile) error
	GetByUserID(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*models.ClientProfile, error)
	GetByUserIDIncludingDeleted(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*models.ClientProfile, error)
	ExistsByPhoneHash(ctx context.Context, tx pgx.Tx, hash string) (bool, error)
	UpdateFields(ctx context.Context, tx pgx.Tx, userID uuid.UUID, updates map[string]any) error
	SoftDelete(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error
	Recover(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error
	AdminGetByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*models.ClientProfile, error)
	AdminList(ctx context.Context, tx pgx.Tx, limit int, cursor *time.Time, search, county, gender *string) ([]models.ClientProfile, error)
}
type AuditLogger interface {
	Log(
		ctx context.Context,
		tx pgx.Tx,
		action string,
		actorID string,
		entityID string,
	) error
}

type ClientService struct {
	DB       *pgxpool.Pool
	Repo     repository.ClientRepository // 👈 REMOVE the '*' here
	UserRepo repository.AuthRepository
	Audit    AuditLogger
}

func NewClientService(
	db *pgxpool.Pool,
	repo repository.ClientRepository, // 👈 REMOVE the '*' here
	userRepo repository.AuthRepository,
	audit AuditLogger,
) *ClientService {
	return &ClientService{
		DB:       db,
		Repo:     repo, // Now this will map correctly without an error
		UserRepo: userRepo,
		Audit:    audit,
	}
}

// Create: client profile creation
// Create: client profile creation
func (s *ClientService) Create(
	ctx context.Context,
	userID uuid.UUID,
	d *dto.CreateClientProfileDTO,
) error {
	if err := d.Validate(); err != nil {
		return err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. 🚨 Check if a profile already exists for this User ID
	// This prevents the "duplicate key" database error and handles Soft Deletes
	existing, err := s.Repo.GetByUserIDIncludingDeleted(ctx, tx, userID)
	if err == nil && existing != nil {
		if existing.IsDeleted {
			return errors.New("you have a deleted profile; please recover it instead of creating a new one")
		}
		return errors.New("profile already exists for this user")
	}

	// 🔐 Normalize and validate inputs
	name, err := utils.SanitizeName(d.FullName)
	if err != nil {
		return err
	}

	phone, err := utils.ValidatePhone(d.PhoneNumber)
	if err != nil {
		return err
	}

	phoneHash := utils.HashSHA256(phone)
	phoneEnc, err := utils.Encrypt(phone)
	if err != nil {
		return err
	}

	// 🚨 Check if phone number is used by ANOTHER user
	phoneExists, err := s.Repo.ExistsByPhoneHash(ctx, tx, phoneHash)
	if err != nil {
		return err
	}
	if phoneExists {
		return errors.New("phone number already registered by another user")
	}

	// 🆔 Optional identity encryption
	var idEnc, idHash, passEnc, passHash *string

	if d.IDNumber != nil && *d.IDNumber != "" {
		e, err := utils.Encrypt(*d.IDNumber)
		if err == nil {
			h := utils.HashSHA256(*d.IDNumber)
			idEnc = &e
			idHash = &h
		}
	}

	if d.PassportNumber != nil && *d.PassportNumber != "" {
		e, err := utils.Encrypt(*d.PassportNumber)
		if err == nil {
			h := utils.HashSHA256(*d.PassportNumber)
			passEnc = &e
			passHash = &h
		}
	}

	profile := models.ClientProfile{
		ID:                      uuid.New(),
		UserID:                  userID,
		FullName:                name,
		IDNumberEncrypted:       idEnc,
		IDNumberHash:            idHash,
		PassportNumberEncrypted: passEnc,
		PassportNumberHash:      passHash,
		PhoneEncrypted:          phoneEnc,
		PhoneHash:               phoneHash,
		Gender:                  d.Gender,
		Nationality:             d.Nationality,
		County:                  d.County,
		Residence:               d.Residence,
	}

	// 2. 📝 Save the profile
	if err := s.Repo.Create(ctx, tx, &profile); err != nil {
		return err
	}

	// 3. ✅ Update user status in the users table
	// This ensures is_profile_complete = TRUE
	if err := s.UserRepo.UpdateProfileStatus(ctx, tx, userID, true); err != nil {
		return err
	}

	// 🧾 Audit log
	if s.Audit != nil {
		_ = s.Audit.Log(ctx, tx,
			"CREATE_CLIENT_PROFILE",
			userID.String(),
			profile.ID.String(),
		)
	}

	return tx.Commit(ctx)
}

// GetMyProfile: owner only
func (s *ClientService) GetMyProfile(ctx context.Context, userID uuid.UUID) (*dto.ClientProfileResponse, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	p, err := s.Repo.GetByUserID(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	// 🔓 Decrypt Phone
	phone, _ := utils.Decrypt(p.PhoneEncrypted)

	// 🔓 Decrypt ID Number if it exists
	var idNum *string
	if p.IDNumberEncrypted != nil {
		dec, err := utils.Decrypt(*p.IDNumberEncrypted)
		if err == nil {
			idNum = &dec
		}
	}

	// 🔓 Decrypt Passport if it exists
	var passNum *string
	if p.PassportNumberEncrypted != nil {
		dec, err := utils.Decrypt(*p.PassportNumberEncrypted)
		if err == nil {
			passNum = &dec
		}
	}

	return &dto.ClientProfileResponse{
		ID:             p.ID.String(),
		FullName:       p.FullName,
		PhoneNumber:    phone,
		IDNumber:       idNum,   // Now assigned!
		PassportNumber: passNum, // Now assigned!
		Gender:         p.Gender,
		Nationality:    p.Nationality,
		County:         p.County,
		Residence:      p.Residence,
		CreatedAt:      p.CreatedAt,
	}, tx.Commit(ctx)
}

// Update: specific fields only
func (s *ClientService) Update(
	ctx context.Context,
	userID uuid.UUID,
	d *dto.UpdateClientProfileDTO,
) error {
	if err := d.Validate(); err != nil {
		return err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	updates := map[string]any{}

	if d.PhoneNumber != nil {
		phone, err := utils.ValidatePhone(*d.PhoneNumber)
		if err != nil {
			return err
		}

		hash := utils.HashSHA256(phone)
		exists, err := s.Repo.ExistsByPhoneHash(ctx, tx, hash)
		if err != nil {
			return err
		}
		if exists {
			return errors.New("phone already in use")
		}

		enc, _ := utils.Encrypt(phone)
		updates["phone_encrypted"] = enc
		updates["phone_hash"] = hash
	}

	if d.Nationality != nil {
		updates["nationality"] = *d.Nationality
	}
	if d.County != nil {
		updates["county"] = *d.County
	}
	if d.Residence != nil {
		updates["residence"] = *d.Residence
	}

	if len(updates) == 0 {
		return errors.New("nothing to update")
	}

	if err := s.Repo.UpdateFields(ctx, tx, userID, updates); err != nil {
		return err
	}

	if s.Audit != nil {
		_ = s.Audit.Log(ctx, tx, "UPDATE_CLIENT_PROFILE", userID.String(), "")
	}

	return tx.Commit(ctx)
}

// Delete: soft delete
func (s *ClientService) Delete(
	ctx context.Context,
	userID uuid.UUID,
) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.Repo.SoftDelete(ctx, tx, userID); err != nil {
		return err
	}

	if s.Audit != nil {
		_ = s.Audit.Log(ctx, tx, "DELETE_CLIENT_PROFILE", userID.String(), "")
	}

	return tx.Commit(ctx)
}

// Recover: within 60 days window
func (s *ClientService) Recover(ctx context.Context, userID uuid.UUID) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 🔴 CRITICAL: Use the method that ignores the is_deleted=FALSE filter
	p, err := s.Repo.GetByUserIDIncludingDeleted(ctx, tx, userID)
	if err != nil {
		// If it hits this, the record truly does not exist in the DB at all
		return err
	}

	if !p.IsDeleted {
		return errors.New("profile is already active")
	}

	// Business rule: 60 days
	if time.Since(*p.DeletedAt) > 60*24*time.Hour {
		return errors.New("recovery window expired")
	}

	// 🟢 Restore it
	if err := s.Repo.Recover(ctx, tx, userID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

//get one admin

func (s *ClientService) AdminGetOne(ctx context.Context, id uuid.UUID) (*dto.AdminClientResponse, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	p, err := s.Repo.AdminGetByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	// 🔓 Decrypt Phone
	phone, _ := utils.Decrypt(p.PhoneEncrypted)

	// 🔓 Decrypt ID Number
	var idNum *string
	if p.IDNumberEncrypted != nil {
		dec, err := utils.Decrypt(*p.IDNumberEncrypted)
		if err == nil {
			idNum = &dec
		}
	}

	// 🔓 Decrypt Passport
	var passNum *string
	if p.PassportNumberEncrypted != nil {
		dec, err := utils.Decrypt(*p.PassportNumberEncrypted)
		if err == nil {
			passNum = &dec
		}
	}

	return &dto.AdminClientResponse{
		ID:             p.ID.String(),
		FullName:       p.FullName,
		PhoneNumber:    phone,
		IDNumber:       idNum,
		PassportNumber: passNum,
		Gender:         p.Gender,
		Nationality:    p.Nationality,
		County:         p.County,
		Residence:      p.Residence,
		FCMToken:       p.FCMToken,
		CreatedAt:      p.CreatedAt,
	}, tx.Commit(ctx)
}

// ADMIN VIEW CLIENTS LIST
func (s *ClientService) AdminList(ctx context.Context, q dto.AdminListClientsQuery) (*dto.AdminListClientsResponse, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q.Normalize()

	// Parse cursor if present
	var cursor *time.Time
	if q.Cursor != nil && *q.Cursor != "" {
		t, err := time.Parse(time.RFC3339, *q.Cursor)
		if err == nil {
			cursor = &t
		}
	}

	list, err := s.Repo.AdminList(ctx, tx, q.Limit, cursor, q.Search, q.County, q.Gender)
	if err != nil {
		return nil, err
	}

	respData := make([]dto.AdminClientResponse, 0, len(list))

	for _, p := range list {
		// 🔓 Decrypt Phone
		phone, _ := utils.Decrypt(p.PhoneEncrypted)

		// 🔓 Decrypt ID
		var idNum *string
		if p.IDNumberEncrypted != nil {
			dec, _ := utils.Decrypt(*p.IDNumberEncrypted)
			idNum = &dec
		}

		// 🔓 Decrypt Passport
		var passNum *string
		if p.PassportNumberEncrypted != nil {
			dec, _ := utils.Decrypt(*p.PassportNumberEncrypted)
			passNum = &dec
		}

		respData = append(respData, dto.AdminClientResponse{
			ID:             p.ID.String(),
			FullName:       p.FullName,
			PhoneNumber:    phone,
			IDNumber:       idNum,
			PassportNumber: passNum,
			Gender:         p.Gender,
			Nationality:    p.Nationality,
			County:         p.County,
			Residence:      p.Residence,
			CreatedAt:      p.CreatedAt,
		})
	}

	var next *time.Time
	if len(list) > 0 && len(list) == q.Limit {
		last := list[len(list)-1].CreatedAt
		next = &last
	}

	return &dto.AdminListClientsResponse{
		Data:       respData,
		NextCursor: next,
	}, tx.Commit(ctx)
}

// /update the is complete in auth table
