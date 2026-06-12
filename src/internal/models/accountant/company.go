package models

import (
	"time"
)

// Company represents the public.companies table
type Company struct {
	CompanyID          string    `json:"company_id"`
	Name               *string   `json:"name"`
	LegalName          *string   `json:"legal_name"`
	TaxIdentifier      *string   `json:"tax_identifier"`
	BaseCurrency       *string   `json:"base_currency"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	Alias              *string   `json:"alias"`
	AccountNumber      *string   `json:"account_number"`
	IfscCode           *string   `json:"ifsc_code"`
	BankName           *string   `json:"bank_name"`
	Branch             *string   `json:"branch"`
	BsrCode            *string   `json:"bsr_code"`
	PanNumber          *string   `json:"pan_number"`
	TypeOfRegistration *string   `json:"type_of_registration"`
	MailingName        *string   `json:"mailing_name"`
	Address            *string   `json:"address"`
	State              *string   `json:"state"`
	Country            *string   `json:"country"`
	PinCode            *string   `json:"pin_code"`
}
