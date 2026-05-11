package services

import (
	"context"
	"fmt"

	"rest_api/internal/core/utils"
	"rest_api/internal/dto"
	"rest_api/internal/repository"

	"github.com/google/uuid"
)

type ContractService interface {
	AutoGenerateAssignmentContracts(ctx context.Context, assignmentID uuid.UUID) error
	GetUserContract(ctx context.Context, assignmentID uuid.UUID, userID uuid.UUID, userRole string) (string, error)
}

type contractService struct {
	repo       repository.ContractRepository
	pdfGen     utils.PDFGenerator
	cloudinary utils.Cloudinary
}

func NewContractService(r repository.ContractRepository, p utils.PDFGenerator, c utils.Cloudinary) ContractService {
	return &contractService{
		repo:       r,
		pdfGen:     p,
		cloudinary: c,
	}
}

// AutoGenerateAssignmentContracts creates both receipts/contracts simultaneously.
func (s *contractService) AutoGenerateAssignmentContracts(ctx context.Context, assignmentID uuid.UUID) error {
	raw, err := s.repo.GetAssignmentDataForContract(ctx, assignmentID)
	if err != nil {
		return fmt.Errorf("failed to fetch contract data: %w", err)
	}

	expiry := raw.StartDate.AddDate(0, raw.DurationMonths, 0)
	nannyNetPay := raw.Salary * 0.90

	// Decrypt using your core/utils/security.go
	empIDDecrypted, _ := utils.Decrypt(raw.EmployerIDNo)
	nanIDDecrypted, _ := utils.Decrypt(raw.NannyIDNo)

	// Employer Receipt DTO
	employerDTO := dto.EmployerAppContractDTO{
		JobRef:         raw.JobRef,
		EmployerName:   raw.EmployerName,
		EmployerIDNo:   empIDDecrypted,
		EmployerPhone:  raw.EmployerPhone,
		EmployerEmail:  raw.EmployerEmail,
		NannyName:      raw.NannyName,
		Salary:         raw.Salary,
		DurationMonths: raw.DurationMonths,
		StartDate:      raw.StartDate,
		ExpiryDate:     expiry,
		Residence:      raw.Residence,
		County:         raw.County,
	}

	// Nanny Deployment DTO
	nannyDTO := dto.NannyAppContractDTO{
		JobRef:         raw.JobRef,
		StartDate:      raw.StartDate,
		EmployerName:   raw.EmployerName,
		NannyName:      raw.NannyName,
		NannyIDNo:      nanIDDecrypted,
		NannyPhone:     raw.NannyPhone,
		GrossSalary:    raw.Salary,
		NetSalary:      nannyNetPay,
		DurationMonths: raw.DurationMonths,
		ExpiryDate:     expiry,
	}

	// Generate PDFs
	empPath, _ := s.pdfGen.GenerateEmployerPDF(employerDTO)
	nanPath, _ := s.pdfGen.GenerateNannyPDF(nannyDTO)

	// Upload to Cloudinary
	empURL, err := s.cloudinary.UploadFile(empPath)
	if err != nil {
		return err
	}
	nanURL, err := s.cloudinary.UploadFile(nanPath)
	if err != nil {
		return err
	}

	return s.repo.UpdateAssignmentContractURLs(ctx, assignmentID, empURL, nanURL)
}

// GetUserContract handles the logic for Nanny, Employer, AND Admin.
func (s *contractService) GetUserContract(ctx context.Context, assignmentID uuid.UUID, userID uuid.UUID, userRole string) (string, error) {
	// 1. If Admin, bypass ownership checks and get the Employer's version by default
	if userRole == "admin" {
		return s.repo.GetContractURL(ctx, assignmentID, uuid.Nil) // Pass Nil to trigger Admin-only logic in repo if needed
	}

	// 2. Otherwise, use the standard ownership firewall
	return s.repo.GetContractURL(ctx, assignmentID, userID)
}
