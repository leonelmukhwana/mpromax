package dto

// CreateJobDTO - Employer posting a new job
type CreateJobDTO struct {
	UserID         string  `json:"user_id"` // Add this!
	EngagementType string  `json:"engagement_type" binding:"required,oneof='Full-time (Live-in)' 'Daily (Come and Go)'"`
	DurationMonths int     `json:"duration_months" binding:"required,min=1"`
	SalaryAmount   float64 `json:"salary_amount" binding:"required,gt=0"`
	Description    string  `json:"description" binding:"required,min=20"`
	County         string  `json:"county" binding:"required"`
	Residence      string  `json:"residence" binding:"required"`
	Requirements   string  `json:"requirements"`
}

// UpdateJobDTO - Allows partial updates (using pointers for pgx compatibility)
type UpdateJobDTO struct {
	EngagementType *string  `json:"engagement_type"`
	DurationMonths *int     `json:"duration_months"`
	SalaryAmount   *float64 `json:"salary_amount"`
	Description    *string  `json:"description"`
	County         *string  `json:"county"`
	Residence      *string  `json:"residence"`
	Requirements   *string  `json:"requirements"`
}

// JobFilterDTO - For Admin's Paginated Search
type JobFilterDTO struct {
	Search   string `form:"search"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1"`
}

// SetDefaults ensures the app doesn't crash if params are missing
func (f *JobFilterDTO) SetDefaults() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 10
	}
}
