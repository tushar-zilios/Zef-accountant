package models

// InventoryEntry represents the public.inventory_entries table
type InventoryEntry struct {
	InventoryEntryID string  `json:"inventory_entry_id"`
	JournalEntryID   string  `json:"journal_entry_id"`
	StockItemID      string  `json:"stock_item_id"`
	Quantity         float64 `json:"quantity"`
	Rate             float64 `json:"rate"`
	Amount           float64 `json:"amount"`
	MovementType     string  `json:"movement_type"`
}
