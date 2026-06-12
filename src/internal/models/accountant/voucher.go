package models

import (
	"time"
)

// Voucher represents the public.vouchers table
type Voucher struct {
	VoucherID      string          `json:"voucher_id"`
	OrganizationID string          `json:"organization_id"`
	FiscalYearID   string          `json:"fiscal_year_id"`
	VoucherTypeID string          `json:"voucher_type_id"`
	VoucherNumber string          `json:"voucher_number"`
	Date          time.Time       `json:"date"`
	Narration     *string         `json:"narration"`
	PostedBy      *string         `json:"posted_by"`
	CreatedAt     *time.Time      `json:"created_at"`
	Entries         []*JournalEntry `json:"entries,omitempty"`
	AllowUnbalanced string          `json:"allow_unbalanced,omitempty"` // set to "true" to skip DR=CR check (e.g. opening entries)
}
