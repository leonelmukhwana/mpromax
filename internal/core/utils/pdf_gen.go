package utils

import (
	"fmt"
	"os"
	"rest_api/internal/dto"

	"github.com/jung-kurt/gofpdf"
)

type PDFGenerator interface {
	GenerateEmployerPDF(data dto.EmployerAppContractDTO) (string, error)
	GenerateNannyPDF(data dto.NannyAppContractDTO) (string, error)
}

type pdfGenerator struct{}

func NewPDFGenerator() PDFGenerator {
	return &pdfGenerator{}
}

func (p *pdfGenerator) GenerateEmployerPDF(d dto.EmployerAppContractDTO) (string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Header
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(190, 10, "SERVICE AGREEMENT: EMPLOYER & PLATFORM")
	pdf.Ln(15)

	// Content Body
	pdf.SetFont("Arial", "", 12)
	content := fmt.Sprintf(
		"Job Reference: %s\nDate Generated: %s\n\n"+
			"EMPLOYER DETAILS:\nName: %s\nID Number: %s\nPhone: %s\nEmail: %s\n\n"+
			"ASSIGNMENT DETAILS:\nNanny Name: %s\nResidence: %s, %s\n"+
			"Monthly Salary: KES %.2f\nDuration: %d Months\n"+
			"Start Date: %s\nExpiry Date: %s\n\n"+
			"This document serves as an official proof of engagement between the employer and the app.",
		d.JobRef, d.StartDate.Format("02 Jan 2006"),
		d.EmployerName, d.EmployerIDNo, d.EmployerPhone, d.EmployerEmail,
		d.NannyName, d.Residence, d.County,
		d.Salary, d.DurationMonths,
		d.StartDate.Format("02 Jan 2006"), d.ExpiryDate.Format("02 Jan 2006"),
	)

	pdf.MultiCell(190, 8, content, "0", "L", false)

	filePath := fmt.Sprintf("tmp/Employer_Contract_%s.pdf", d.JobRef)
	os.MkdirAll("tmp", os.ModePerm)
	err := pdf.OutputFileAndClose(filePath)
	return filePath, err
}

func (p *pdfGenerator) GenerateNannyPDF(d dto.NannyAppContractDTO) (string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(190, 10, "DEPLOYMENT ORDER: NANNY & PLATFORM")
	pdf.Ln(15)

	pdf.SetFont("Arial", "", 12)
	content := fmt.Sprintf(
		"Job Reference: %s\nAssignment Date: %s\n\n"+
			"NANNY DETAILS:\nName: %s\nID Number: %s\nPhone: %s\n\n"+
			"DEPLOYMENT TERMS:\nEmployer Name: %s\n"+
			"Contract Duration: %d Months\nExpiry Date: %s\n\n"+
			"COMPENSATION:\nGross Salary: KES %.2f\nNet Amount Payable (90%%): KES %.2f\n\n"+
			"By receiving this deployment order, you agree to the platform's terms of conduct.",
		d.JobRef, d.StartDate.Format("02 Jan 2006"),
		d.NannyName, d.NannyIDNo, d.NannyPhone,
		d.EmployerName, d.DurationMonths, d.ExpiryDate.Format("02 Jan 2006"),
		d.GrossSalary, d.NetSalary,
	)

	pdf.MultiCell(190, 8, content, "0", "L", false)

	filePath := fmt.Sprintf("tmp/Nanny_Contract_%s.pdf", d.JobRef)
	os.MkdirAll("tmp", os.ModePerm)
	err := pdf.OutputFileAndClose(filePath)
	return filePath, err
}
