package models

// Ledger represents the public.ledgers table
type Ledger struct {
	LedgerID           string  `json:"ledger_id"`
	OrganizationID     string  `json:"organization_id"`
	GroupID            string  `json:"group_id"`
	Name               string  `json:"name"`
	Currency           string  `json:"currency"`
	OpeningBalance     float64 `json:"opening_balance"`
	OpeningBalanceType string  `json:"opening_balance_type"`
	IsActive           *bool   `json:"is_active"`
	// Tally-style per-ledger metadata (banking, address, tax)
	LegalName     *string `json:"legal_name,omitempty"`
	TaxIdentifier *string `json:"tax_identifier,omitempty"`
	BaseCurrency  *string `json:"base_currency,omitempty"`
	Alias         *string `json:"alias,omitempty"`
	AccountNo     *string `json:"account_no,omitempty"`
	IfsCode       *string `json:"ifs_code,omitempty"`
	BankName      *string `json:"bank_name,omitempty"`
	Branch        *string `json:"branch,omitempty"`
	BsrCode       *string `json:"bsr_code,omitempty"`
	PanItNo       *string `json:"pan_it_no,omitempty"`
	TypeOfReg     *string `json:"type_of_reg,omitempty"`
	MailingName   *string `json:"mailing_name,omitempty"`
	Address       *string `json:"address,omitempty"`
	State         *string `json:"state,omitempty"`
	Country       *string `json:"country,omitempty"`
	PinCode       *string `json:"pin_code,omitempty"`
	GSTType       *string `json:"gst_type,omitempty"`
}

// PatchLedgerInput represents the fields that can be partially updated on a ledger
type PatchLedgerInput struct {
	OrganizationID     *string  `json:"organization_id,omitempty"`
	GroupID            *string  `json:"group_id,omitempty"`
	Name               *string  `json:"name,omitempty"`
	Currency           *string  `json:"currency,omitempty"`
	OpeningBalance     *float64 `json:"opening_balance,omitempty"`
	OpeningBalanceType *string  `json:"opening_balance_type,omitempty"`
	IsActive           *bool    `json:"is_active,omitempty"`
	LegalName          *string  `json:"legal_name,omitempty"`
	TaxIdentifier      *string  `json:"tax_identifier,omitempty"`
	BaseCurrency       *string  `json:"base_currency,omitempty"`
	Alias              *string  `json:"alias,omitempty"`
	AccountNo          *string  `json:"account_no,omitempty"`
	IfsCode            *string  `json:"ifs_code,omitempty"`
	BankName           *string  `json:"bank_name,omitempty"`
	Branch             *string  `json:"branch,omitempty"`
	BsrCode            *string  `json:"bsr_code,omitempty"`
	PanItNo            *string  `json:"pan_it_no,omitempty"`
	TypeOfReg          *string  `json:"type_of_reg,omitempty"`
	MailingName        *string  `json:"mailing_name,omitempty"`
	Address            *string  `json:"address,omitempty"`
	State              *string  `json:"state,omitempty"`
	Country            *string  `json:"country,omitempty"`
	PinCode            *string  `json:"pin_code,omitempty"`
	GSTType            *string  `json:"gst_type,omitempty"`
}
