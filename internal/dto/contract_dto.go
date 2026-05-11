package dto

import "time"

// ContractSourceData is the raw data pulled from the DB join.
// It is used internally by the repository and service.
type ContractSourceData struct {
	JobRef         string
	Salary         float64
	DurationMonths int
	StartDate      time.Time
	Residence      string
	County         string
	EmployerName   string
	EmployerIDNo   string // Encrypted string from DB
	EmployerPhone  string
	EmployerEmail  string
	NannyName      string
	NannyIDNo      string // Encrypted string from DB
	NannyPhone     string
}

// EmployerAppContractDTO is the clean data for the Employer's PDF.
type EmployerAppContractDTO struct {
	JobRef         string
	EmployerName   string
	EmployerIDNo   string // Decrypted
	EmployerPhone  string
	EmployerEmail  string
	NannyName      string
	Salary         float64
	DurationMonths int
	StartDate      time.Time
	ExpiryDate     time.Time
	Residence      string
	County         string
}

// NannyAppContractDTO is the clean data for the Nanny's PDF.
type NannyAppContractDTO struct {
	JobRef         string
	StartDate      time.Time // The Assignment Date
	EmployerName   string
	NannyName      string
	NannyIDNo      string // Decrypted
	NannyPhone     string
	GrossSalary    float64
	NetSalary      float64 // The 90% amount
	DurationMonths int
	ExpiryDate     time.Time
}
