package models

// OrganizationToCompany represents the public.organization_to_company mapping table
type OrganizationToCompany struct {
	MappingID      string  `json:"mapping_id"`
	OrganizationID string  `json:"organization_id"`
	CompanyID      *string `json:"company_id"`
}
