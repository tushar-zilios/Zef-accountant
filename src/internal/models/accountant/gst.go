package models

import "time"

// GSTEntry represents a calculated GST report line item for GSTR1/2/3B
type GSTEntry struct {
	VoucherNumber string    `json:"voucher_number"`
	Date          time.Time `json:"date"`
	PartyName     string    `json:"party_name"`
	VoucherType   string    `json:"voucher_type"`
	Section       string    `json:"section"` // "OUTPUT" for sales, "INPUT" for purchases
	TaxableAmount float64   `json:"taxable_amount"`
	CGST          float64   `json:"cgst"`
	SGST          float64   `json:"sgst"`
	IGST          float64   `json:"igst"`
	TotalTax      float64   `json:"total_tax"`
	GrandTotal    float64   `json:"grand_total"`
}

// GSTR3BSummary is the aggregated monthly return summary
type GSTR3BSummary struct {
	// Outward supplies
	OutputTaxableAmount float64 `json:"output_taxable_amount"`
	OutputCGST          float64 `json:"output_cgst"`
	OutputSGST          float64 `json:"output_sgst"`
	OutputIGST          float64 `json:"output_igst"`
	OutputTotalTax      float64 `json:"output_total_tax"`

	// Input tax credit
	InputTaxableAmount float64 `json:"input_taxable_amount"`
	InputCGST          float64 `json:"input_cgst"`
	InputSGST          float64 `json:"input_sgst"`
	InputIGST          float64 `json:"input_igst"`
	InputTotalTax      float64 `json:"input_total_tax"`

	// Net payable
	NetCGST float64 `json:"net_cgst"`
	NetSGST float64 `json:"net_sgst"`
	NetIGST float64 `json:"net_igst"`
	NetTax  float64 `json:"net_tax"`

	Entries []*GSTEntry `json:"entries"`
}
