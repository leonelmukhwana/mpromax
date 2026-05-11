package services

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"rest_api/internal/models"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

type VerificationRepository interface {
	CheckEligibility(ctx context.Context, userID uuid.UUID) (bool, error)
	UpsertVerification(ctx context.Context, nannyID uuid.UUID, idUrl, selfieUrl, idPubID, selfiePubID string) error

	GetByNannyID(ctx context.Context, nannyID uuid.UUID) (models.NannyVerification, error)

	// 👈 ADD THIS: This fixes the second error
	GetAllVerifications(ctx context.Context) ([]models.NannyVerification, error)
	AdminArchiveUpdate(ctx context.Context, nannyID uuid.UUID) error
}

type VerificationService struct {
	Repo       VerificationRepository
	Cloudinary *cloudinary.Cloudinary
}

func NewVerificationService(repo VerificationRepository, cld *cloudinary.Cloudinary) *VerificationService {
	return &VerificationService{
		Repo:       repo,
		Cloudinary: cld,
	}
}

// ProcessVerification handles the entire logic: Eligibility -> Resize -> Upload -> Save
func (s *VerificationService) ProcessVerification(ctx context.Context, nannyID uuid.UUID, idReader, selfieReader io.Reader) error {
	// 1. GATEKEEPER: Check if user is a nanny and profile is complete
	isEligible, err := s.Repo.CheckEligibility(ctx, nannyID)
	if err != nil {
		return err
	}
	if !isEligible {
		return errors.New("nanny profile must be complete before uploading documents")
	}

	// 2. PROCESS & UPLOAD ID CARD
	idURL, idPublicID, err := s.resizeAndUpload(ctx, nannyID, idReader, "id_card")
	if err != nil {
		return err
	}

	// 3. PROCESS & UPLOAD SELFIE
	selfieURL, selfiePublicID, err := s.resizeAndUpload(ctx, nannyID, selfieReader, "selfie")
	if err != nil {
		return err
	}

	// 4. ATOMIC DB UPSERT
	return s.Repo.UpsertVerification(ctx, nannyID, idURL, selfieURL, idPublicID, selfiePublicID)
}

// resizeAndUpload is a private helper to reduce code duplication
func (s *VerificationService) resizeAndUpload(ctx context.Context, nannyID uuid.UUID, reader io.Reader, fileType string) (string, string, error) {
	// Decode the image
	src, _, err := image.Decode(reader)
	if err != nil {
		return "", "", errors.New("failed to decode image: " + err.Error())
	}

	// Resize: Max width 1024px, maintain aspect ratio
	// This makes the file tiny (~100-200kb) before it even hits the internet
	dst := imaging.Resize(src, 1024, 0, imaging.Lanczos)

	// Encode to JPEG buffer with 75% quality (excellent for text/faces)
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, dst, &jpeg.Options{Quality: 75})
	if err != nil {
		return "", "", err
	}

	// Upload to Cloudinary
	uploadRes, err := s.Cloudinary.Upload.Upload(ctx, buf, uploader.UploadParams{
		Folder:   "nanny_verifications",
		PublicID: "nanny_" + nannyID.String() + "_" + fileType,
	})
	if err != nil {
		return "", "", errors.New("cloudinary upload failed: " + err.Error())
	}

	return uploadRes.SecureURL, uploadRes.PublicID, nil
}

// PurgeCloudinaryAssets handles the actual deletion from Cloudinary
func (s *VerificationService) PurgeCloudinaryAssets(ctx context.Context, nannyID uuid.UUID) error {
	// 1. Get the current records from DB.
	// 'data' is now already a models.NannyVerification struct.
	data, err := s.Repo.GetByNannyID(ctx, nannyID)
	if err != nil {
		return err
	}

	// 2. Delete ID Card from Cloudinary using 'data' directly
	if data.IDPublicID != "" {
		_, err := s.Cloudinary.Upload.Destroy(ctx, uploader.DestroyParams{
			PublicID: data.IDPublicID,
		})
		if err != nil {
			return err
		}
	}

	// 3. Delete Selfie from Cloudinary
	if data.SelfiePublicID != "" {
		// We don't necessarily need to block the whole process if one delete fails,
		// but it's good practice to handle the error or log it.
		_, _ = s.Cloudinary.Upload.Destroy(ctx, uploader.DestroyParams{
			PublicID: data.SelfiePublicID,
		})
	}

	// 4. Update the DB to mark as 'ARCHIVED'
	return s.Repo.AdminArchiveUpdate(ctx, nannyID)
}
