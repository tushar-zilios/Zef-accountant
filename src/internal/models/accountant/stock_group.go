package models

// StockGroup represents the public.stock_groups table
type StockGroup struct {
	StockGroupID   string  `json:"stock_group_id"`
	OrganizationID string  `json:"organization_id"`
	Name           string  `json:"name"`
	ParentID       *string `json:"parent_id"`
	HSNCode        string  `json:"hsn_code"`
	GSTRate        float64 `json:"gst_rate"`
}
