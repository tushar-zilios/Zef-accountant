package models

import "time"

type EInvoice struct {
	EInvoiceID     string     `json:"einvoice_id"`
	VoucherID      string     `json:"voucher_id"`
	OrganizationID string     `json:"organization_id"`
	FiscalYearID   string     `json:"fiscal_year_id"`
	IRN            string     `json:"irn"`
	AckNo          string     `json:"ack_no"`
	AckDate        time.Time  `json:"ack_date"`
	SellerGSTIN    string     `json:"seller_gstin"`
	BuyerGSTIN     string     `json:"buyer_gstin"`
	InvoiceNo      string     `json:"invoice_no"`
	InvoiceDate    string     `json:"invoice_date"`
	TotalValue     float64    `json:"total_value"`
	CGST           float64    `json:"cgst"`
	SGST           float64    `json:"sgst"`
	IGST           float64    `json:"igst"`
	Status         string     `json:"status"`
	CreatedAt      *time.Time `json:"created_at"`
}

type EWayBill struct {
	EWayBillID      string     `json:"eway_bill_id"`
	VoucherID       string     `json:"voucher_id"`
	OrganizationID  string     `json:"organization_id"`
	FiscalYearID    string     `json:"fiscal_year_id"`
	EWBNo           string     `json:"ewb_no"`
	EWBDate         time.Time  `json:"ewb_date"`
	ValidUpto       string     `json:"valid_upto"`
	SellerGSTIN     string     `json:"seller_gstin"`
	BuyerGSTIN      string     `json:"buyer_gstin"`
	TransporterID   string     `json:"transporter_id"`
	TransporterName string     `json:"transporter_name"`
	VehicleNo       string     `json:"vehicle_no"`
	VehicleType     string     `json:"vehicle_type"`
	DispatchFrom    string     `json:"dispatch_from"`
	ShipTo          string     `json:"ship_to"`
	Distance        int        `json:"distance_km"`
	TotalValue      float64    `json:"total_value"`
	Status          string     `json:"status"`
	CreatedAt       *time.Time `json:"created_at"`
}
