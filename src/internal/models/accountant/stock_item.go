package models

// StockItem represents the public.stock_items table
type StockItem struct {
	StockItemID      string  `json:"stock_item_id"`
	GroupID          *string `json:"group_id"`
	Name             string  `json:"name"`
	UnitOfMeasure    string  `json:"unit_of_measure"`
	OpeningQty       float64 `json:"opening_qty"`
	OpeningValuation float64 `json:"opening_valuation"`
	CostingMethod    string  `json:"costing_method"`
	HSNCode          string  `json:"hsn_code"`
	GSTRate          float64 `json:"gst_rate"`
}

type StockSummaryItem struct {
	StockItemID   string  `json:"stock_item_id"`
	Name          string  `json:"name"`
	UnitOfMeasure string  `json:"unit_of_measure"`
	ClosingQty    float64 `json:"closing_qty"`
	AvgRate       float64 `json:"avg_rate"`
	Valuation     float64 `json:"valuation"`
}

