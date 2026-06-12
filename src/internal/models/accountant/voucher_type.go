package models

// VoucherType represents the public.voucher_types table
type VoucherType struct {
	VoucherTypeID  string  `json:"voucher_type_id"`
	OrganizationID string  `json:"organization_id"`
	Name           string  `json:"name"`
	Prefix         *string `json:"prefix"`
	IsAutoNumbered *bool   `json:"is_auto_numbered"`
	IsReserved     *bool   `json:"is_reserved"`
	ParentBaseType *string `json:"parent_base_type"`
}
