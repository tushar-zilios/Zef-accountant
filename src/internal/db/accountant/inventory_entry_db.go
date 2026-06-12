package accountant

import (
	"context"
	"accountant/src/internal/db"
	"accountant/src/internal/models/accountant"
)

// --- InventoryEntry Actions ---

func CreateInventoryEntry(ctx context.Context, i *models.InventoryEntry) error {
	query := `
		INSERT INTO public.inventory_entries (journal_entry_id, stock_item_id, quantity, rate, amount, movement_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING inventory_entry_id
	`
	return db.GetPool().QueryRow(ctx, query, i.JournalEntryID, i.StockItemID, i.Quantity, i.Rate, i.Amount, i.MovementType).
		Scan(&i.InventoryEntryID)
}

func GetInventoryEntryByID(ctx context.Context, id string) (*models.InventoryEntry, error) {
	query := `
		SELECT inventory_entry_id, journal_entry_id, stock_item_id, quantity, rate, amount, movement_type
		FROM public.inventory_entries WHERE inventory_entry_id = $1
	`
	i := &models.InventoryEntry{}
	err := db.GetPool().QueryRow(ctx, query, id).
		Scan(&i.InventoryEntryID, &i.JournalEntryID, &i.StockItemID, &i.Quantity, &i.Rate, &i.Amount, &i.MovementType)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func UpdateInventoryEntry(ctx context.Context, i *models.InventoryEntry) error {
	query := `
		UPDATE public.inventory_entries
		SET journal_entry_id = $1, stock_item_id = $2, quantity = $3, rate = $4, amount = $5, movement_type = $6
		WHERE inventory_entry_id = $7
	`
	_, err := db.GetPool().Exec(ctx, query, i.JournalEntryID, i.StockItemID, i.Quantity, i.Rate, i.Amount, i.MovementType, i.InventoryEntryID)
	return err
}

func DeleteInventoryEntry(ctx context.Context, id string) error {
	query := `DELETE FROM public.inventory_entries WHERE inventory_entry_id = $1`
	_, err := db.GetPool().Exec(ctx, query, id)
	return err
}

func ListInventoryEntries(ctx context.Context, limit, offset int) ([]*models.InventoryEntry, error) {
	query := `
		SELECT inventory_entry_id, journal_entry_id, stock_item_id, quantity, rate, amount, movement_type
		FROM public.inventory_entries LIMIT $1 OFFSET $2
	`
	rows, err := db.GetPool().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.InventoryEntry
	for rows.Next() {
		i := &models.InventoryEntry{}
		if err := rows.Scan(&i.InventoryEntryID, &i.JournalEntryID, &i.StockItemID, &i.Quantity, &i.Rate, &i.Amount, &i.MovementType); err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, nil
}

func GetInventoryEntriesByJournalEntryID(ctx context.Context, journalEntryID string) ([]*models.InventoryEntry, error) {
	query := `
		SELECT inventory_entry_id, journal_entry_id, stock_item_id, quantity, rate, amount, movement_type
		FROM public.inventory_entries WHERE journal_entry_id = $1
	`
	rows, err := db.GetPool().Query(ctx, query, journalEntryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.InventoryEntry
	for rows.Next() {
		i := &models.InventoryEntry{}
		if err := rows.Scan(&i.InventoryEntryID, &i.JournalEntryID, &i.StockItemID, &i.Quantity, &i.Rate, &i.Amount, &i.MovementType); err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, nil
}
