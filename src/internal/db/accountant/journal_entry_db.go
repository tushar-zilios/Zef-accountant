package accountant

import (
	"context"
	"accountant/src/internal/db"
	"accountant/src/internal/models/accountant"
)

// --- JournalEntry Actions ---

func CreateJournalEntry(ctx context.Context, j *models.JournalEntry) error {
	query := `
		INSERT INTO public.journal_entries (voucher_id, ledger_id, debit, credit, currency_rate, narration, sequence_order)
		VALUES ($1, $2, COALESCE($3, 0.0000), COALESCE($4, 0.0000), COALESCE($5, 1.000000), $6, $7)
		RETURNING journal_entry_id, debit, credit, currency_rate
	`
	return db.GetPool().QueryRow(ctx, query, j.VoucherID, j.LedgerID, j.Debit, j.Credit, j.CurrencyRate, j.Narration, j.SequenceOrder).
		Scan(&j.JournalEntryID, &j.Debit, &j.Credit, &j.CurrencyRate)
}

func GetJournalEntryByID(ctx context.Context, id string) (*models.JournalEntry, error) {
	query := `
		SELECT journal_entry_id, voucher_id, ledger_id, debit, credit, currency_rate, narration, sequence_order
		FROM public.journal_entries WHERE journal_entry_id = $1
	`
	j := &models.JournalEntry{}
	err := db.GetPool().QueryRow(ctx, query, id).
		Scan(&j.JournalEntryID, &j.VoucherID, &j.LedgerID, &j.Debit, &j.Credit, &j.CurrencyRate, &j.Narration, &j.SequenceOrder)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func UpdateJournalEntry(ctx context.Context, j *models.JournalEntry) error {
	query := `
		UPDATE public.journal_entries
		SET voucher_id = $1, ledger_id = $2, debit = $3, credit = $4, currency_rate = $5, narration = $6, sequence_order = $7
		WHERE journal_entry_id = $8
	`
	_, err := db.GetPool().Exec(ctx, query, j.VoucherID, j.LedgerID, j.Debit, j.Credit, j.CurrencyRate, j.Narration, j.SequenceOrder, j.JournalEntryID)
	return err
}

func DeleteJournalEntry(ctx context.Context, id string) error {
	query := `DELETE FROM public.journal_entries WHERE journal_entry_id = $1`
	_, err := db.GetPool().Exec(ctx, query, id)
	return err
}

func ListJournalEntries(ctx context.Context, limit, offset int) ([]*models.JournalEntry, error) {
	query := `
		SELECT journal_entry_id, voucher_id, ledger_id, debit, credit, currency_rate, narration, sequence_order
		FROM public.journal_entries LIMIT $1 OFFSET $2
	`
	rows, err := db.GetPool().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.JournalEntry
	for rows.Next() {
		j := &models.JournalEntry{}
		if err := rows.Scan(&j.JournalEntryID, &j.VoucherID, &j.LedgerID, &j.Debit, &j.Credit, &j.CurrencyRate, &j.Narration, &j.SequenceOrder); err != nil {
			return nil, err
		}
		list = append(list, j)
	}
	return list, nil
}

func GetJournalEntriesByVoucherID(ctx context.Context, voucherID string) ([]*models.JournalEntry, error) {
	query := `
		SELECT journal_entry_id, voucher_id, ledger_id, debit, credit, currency_rate, narration, sequence_order
		FROM public.journal_entries WHERE voucher_id = $1
		ORDER BY sequence_order ASC
	`
	rows, err := db.GetPool().Query(ctx, query, voucherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.JournalEntry
	for rows.Next() {
		j := &models.JournalEntry{}
		if err := rows.Scan(&j.JournalEntryID, &j.VoucherID, &j.LedgerID, &j.Debit, &j.Credit, &j.CurrencyRate, &j.Narration, &j.SequenceOrder); err != nil {
			return nil, err
		}
		list = append(list, j)
	}
	return list, nil
}

func GetJournalEntriesByLedgerID(ctx context.Context, ledgerID string) ([]*models.JournalEntry, error) {
	query := `
		SELECT journal_entry_id, voucher_id, ledger_id, debit, credit, currency_rate, narration, sequence_order
		FROM public.journal_entries WHERE ledger_id = $1
	`
	rows, err := db.GetPool().Query(ctx, query, ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.JournalEntry
	for rows.Next() {
		j := &models.JournalEntry{}
		if err := rows.Scan(&j.JournalEntryID, &j.VoucherID, &j.LedgerID, &j.Debit, &j.Credit, &j.CurrencyRate, &j.Narration, &j.SequenceOrder); err != nil {
			return nil, err
		}
		list = append(list, j)
	}
	return list, nil
}
