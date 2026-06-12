package models

type Godown struct {
	GodownID       string `json:"godown_id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Type           string `json:"type"` // warehouse, transit, location
	Address        string `json:"address"`
	Description    string `json:"description"`
	CreatedAt      string `json:"created_at,omitempty"`
}

type GodownStockItem struct {
	GodownID      string  `json:"godown_id"`
	GodownName    string  `json:"godown_name"`
	StockItemID   string  `json:"stock_item_id"`
	StockItemName string  `json:"stock_item_name"`
	UnitOfMeasure string  `json:"unit_of_measure"`
	Quantity      float64 `json:"quantity"`
	AvgRate       float64 `json:"avg_rate"`
	Valuation     float64 `json:"valuation"`
}

type StockTransfer struct {
	TransferID     string  `json:"transfer_id"`
	OrganizationID string  `json:"organization_id"`
	FromGodownID   *string `json:"from_godown_id"`
	ToGodownID     *string `json:"to_godown_id"`
	StockItemID    string  `json:"stock_item_id"`
	Quantity       float64 `json:"quantity"`
	Rate           float64 `json:"rate"`
	TransferDate   string  `json:"transfer_date"`
	Remarks        string  `json:"remarks"`
	CreatedAt      string  `json:"created_at,omitempty"`
	// Joined fields
	FromGodownName  string `json:"from_godown_name,omitempty"`
	ToGodownName    string `json:"to_godown_name,omitempty"`
	StockItemName   string `json:"stock_item_name,omitempty"`
	UnitOfMeasure   string `json:"unit_of_measure,omitempty"`
}
