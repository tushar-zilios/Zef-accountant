package models

import (
	"time"
)

// FiscalYear represents the public.fiscal_years table
type FiscalYear struct {
	FiscalYearID string    `json:"fiscal_year_id"`
	OrganizationID string    `json:"organization_id"`
	YearLabel    string    `json:"year_label"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	IsClosed     *bool     `json:"is_closed"`
}
