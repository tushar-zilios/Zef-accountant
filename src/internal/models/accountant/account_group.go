package models

// AccountGroup represents the public.account_groups table
type AccountGroup struct {
	AccountGroupID string       `json:"account_group_id"`
	OrganizationID string       `json:"organization_id"`
	ParentID       *string      `json:"parent_id"`
	Name           string       `json:"name"`
	Classification AccountClass `json:"classification"`
	IsReserved     *bool        `json:"is_reserved"`
}

// GroupBalance represents the calculated balance of an AccountGroup
type GroupBalance struct {
	AccountGroupID string  `json:"account_group_id"`
	TotalDebit     float64 `json:"total_debit"`
	TotalCredit    float64 `json:"total_credit"`
	NetBalanceDR   float64 `json:"net_balance_dr"`
	NetBalanceCR   float64 `json:"net_balance_cr"`
	Balance        float64 `json:"balance"` // Classification-aware balance (positive for normal, negative for abnormal)
}

