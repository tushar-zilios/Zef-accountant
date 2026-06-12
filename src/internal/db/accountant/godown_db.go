package accountant

import (
	"context"
	"accountant/src/internal/db"
	models "accountant/src/internal/models/accountant"
)

// --- Godown CRUD ---

func CreateGodown(ctx context.Context, g *models.Godown) error {
	query := `
		INSERT INTO public.godowns (organization_id, name, type, address, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING godown_id, created_at
	`
	return db.GetPool().QueryRow(ctx, query, g.OrganizationID, g.Name, g.Type, g.Address, g.Description).
		Scan(&g.GodownID, &g.CreatedAt)
}

func GetGodownByID(ctx context.Context, id string) (*models.Godown, error) {
	query := `SELECT godown_id, organization_id, name, type, address, description, created_at FROM public.godowns WHERE godown_id = $1`
	g := &models.Godown{}
	err := db.GetPool().QueryRow(ctx, query, id).
		Scan(&g.GodownID, &g.OrganizationID, &g.Name, &g.Type, &g.Address, &g.Description, &g.CreatedAt)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func UpdateGodown(ctx context.Context, g *models.Godown) error {
	query := `UPDATE public.godowns SET name=$1, type=$2, address=$3, description=$4 WHERE godown_id=$5`
	_, err := db.GetPool().Exec(ctx, query, g.Name, g.Type, g.Address, g.Description, g.GodownID)
	return err
}

func DeleteGodown(ctx context.Context, id string) error {
	_, err := db.GetPool().Exec(ctx, `DELETE FROM public.godowns WHERE godown_id=$1`, id)
	return err
}

func GetGodownsByOrganizationID(ctx context.Context, orgID string) ([]*models.Godown, error) {
	query := `SELECT godown_id, organization_id, name, type, address, description, created_at FROM public.godowns WHERE organization_id=$1 ORDER BY name`
	rows, err := db.GetPool().Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var godowns []*models.Godown
	for rows.Next() {
		g := &models.Godown{}
		if err := rows.Scan(&g.GodownID, &g.OrganizationID, &g.Name, &g.Type, &g.Address, &g.Description, &g.CreatedAt); err != nil {
			return nil, err
		}
		godowns = append(godowns, g)
	}
	return godowns, nil
}

// --- Godown Stock Summary ---

// GetGodownStockSummary returns per-godown, per-item quantities computed from inventory_entries.
func GetGodownStockSummary(ctx context.Context, orgID string) ([]*models.GodownStockItem, error) {
	query := `
		SELECT
			g.godown_id,
			g.name AS godown_name,
			si.stock_item_id,
			si.name AS stock_item_name,
			si.unit_of_measure,
			SUM(CASE WHEN ie.movement_type = 'IN' THEN ie.quantity ELSE -ie.quantity END) AS quantity,
			CASE WHEN SUM(CASE WHEN ie.movement_type = 'IN' THEN ie.quantity ELSE 0 END) > 0
				THEN SUM(CASE WHEN ie.movement_type = 'IN' THEN ie.amount ELSE 0 END) /
				     SUM(CASE WHEN ie.movement_type = 'IN' THEN ie.quantity ELSE 0 END)
				ELSE 0
			END AS avg_rate,
			SUM(CASE WHEN ie.movement_type = 'IN' THEN ie.amount ELSE -ie.amount END) AS valuation
		FROM public.inventory_entries ie
		JOIN public.godowns g ON ie.godown_id = g.godown_id
		JOIN public.stock_items si ON ie.stock_item_id = si.stock_item_id
		WHERE g.organization_id = $1
		GROUP BY g.godown_id, g.name, si.stock_item_id, si.name, si.unit_of_measure
		ORDER BY g.name, si.name
	`
	rows, err := db.GetPool().Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*models.GodownStockItem
	for rows.Next() {
		item := &models.GodownStockItem{}
		if err := rows.Scan(&item.GodownID, &item.GodownName, &item.StockItemID, &item.StockItemName, &item.UnitOfMeasure, &item.Quantity, &item.AvgRate, &item.Valuation); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, nil
}

// --- Stock Transfers ---

func CreateStockTransfer(ctx context.Context, t *models.StockTransfer) error {
	query := `
		INSERT INTO public.stock_transfers (organization_id, from_godown_id, to_godown_id, stock_item_id, quantity, rate, transfer_date, remarks)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING transfer_id, created_at
	`
	return db.GetPool().QueryRow(ctx, query, t.OrganizationID, t.FromGodownID, t.ToGodownID, t.StockItemID, t.Quantity, t.Rate, t.TransferDate, t.Remarks).
		Scan(&t.TransferID, &t.CreatedAt)
}

func GetStockTransfersByOrganizationID(ctx context.Context, orgID string) ([]*models.StockTransfer, error) {
	query := `
		SELECT
			st.transfer_id, st.organization_id,
			st.from_godown_id, st.to_godown_id,
			st.stock_item_id, st.quantity, st.rate,
			st.transfer_date::text, st.remarks, st.created_at::text,
			COALESCE(fg.name, ''), COALESCE(tg.name, ''),
			si.name, si.unit_of_measure
		FROM public.stock_transfers st
		LEFT JOIN public.godowns fg ON st.from_godown_id = fg.godown_id
		LEFT JOIN public.godowns tg ON st.to_godown_id = tg.godown_id
		JOIN public.stock_items si ON st.stock_item_id = si.stock_item_id
		WHERE st.organization_id = $1
		ORDER BY st.created_at DESC
	`
	rows, err := db.GetPool().Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*models.StockTransfer
	for rows.Next() {
		t := &models.StockTransfer{}
		if err := rows.Scan(&t.TransferID, &t.OrganizationID, &t.FromGodownID, &t.ToGodownID, &t.StockItemID, &t.Quantity, &t.Rate, &t.TransferDate, &t.Remarks, &t.CreatedAt, &t.FromGodownName, &t.ToGodownName, &t.StockItemName, &t.UnitOfMeasure); err != nil {
			return nil, err
		}
		results = append(results, t)
	}
	return results, nil
}

func DeleteStockTransfer(ctx context.Context, id string) error {
	_, err := db.GetPool().Exec(ctx, `DELETE FROM public.stock_transfers WHERE transfer_id=$1`, id)
	return err
}
