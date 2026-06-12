package models

// JournalEntry represents the public.journal_entries table
type JournalEntry struct {
	JournalEntryID string  `json:"journal_entry_id"`
	VoucherID      string  `json:"voucher_id"`
	LedgerID       string  `json:"ledger_id"`
	Debit          float64 `json:"debit"`
	Credit         float64 `json:"credit"`
	CurrencyRate   float64 `json:"currency_rate"`
	Narration      *string `json:"narration"`
	SequenceOrder  int     `json:"sequence_order"`
}
