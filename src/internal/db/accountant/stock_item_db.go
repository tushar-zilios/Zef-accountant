package accountant

import (
	"context"
	"accountant/src/internal/db"
	"accountant/src/internal/models/accountant"
)

// --- StockItem Actions ---

func CreateStockItem(ctx context.Context, s *models.StockItem) error {
	query := `
		INSERT INTO public.stock_items (group_id, name, unit_of_measure, opening_qty, opening_valuation, costing_method, hsn_code, gst_rate)
		VALUES ($1, $2, $3, COALESCE($4, 0.0000), COALESCE($5, 0.0000), COALESCE($6, 'FIFO'), COALESCE($7, ''), COALESCE($8, 0))
		RETURNING stock_item_id, opening_qty, opening_valuation, costing_method, hsn_code, gst_rate
	`
	return db.GetPool().QueryRow(ctx, query, s.GroupID, s.Name, s.UnitOfMeasure, s.OpeningQty, s.OpeningValuation, s.CostingMethod, s.HSNCode, s.GSTRate).
		Scan(&s.StockItemID, &s.OpeningQty, &s.OpeningValuation, &s.CostingMethod, &s.HSNCode, &s.GSTRate)
}

func GetStockItemByID(ctx context.Context, id string) (*models.StockItem, error) {
	query := `
		SELECT stock_item_id, group_id, name, unit_of_measure, opening_qty, opening_valuation, costing_method,
		       COALESCE(hsn_code, ''), COALESCE(gst_rate, 0)
		FROM public.stock_items WHERE stock_item_id = $1
	`
	s := &models.StockItem{}
	err := db.GetPool().QueryRow(ctx, query, id).
		Scan(&s.StockItemID, &s.GroupID, &s.Name, &s.UnitOfMeasure, &s.OpeningQty, &s.OpeningValuation, &s.CostingMethod, &s.HSNCode, &s.GSTRate)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func UpdateStockItem(ctx context.Context, s *models.StockItem) error {
	query := `
		UPDATE public.stock_items
		SET group_id = $1, name = $2, unit_of_measure = $3, opening_qty = $4, opening_valuation = $5,
		    costing_method = $6, hsn_code = $7, gst_rate = $8
		WHERE stock_item_id = $9
	`
	_, err := db.GetPool().Exec(ctx, query, s.GroupID, s.Name, s.UnitOfMeasure, s.OpeningQty, s.OpeningValuation, s.CostingMethod, s.HSNCode, s.GSTRate, s.StockItemID)
	return err
}

func DeleteStockItem(ctx context.Context, id string) error {
	query := `DELETE FROM public.stock_items WHERE stock_item_id = $1`
	_, err := db.GetPool().Exec(ctx, query, id)
	return err
}

func ListStockItems(ctx context.Context, limit, offset int) ([]*models.StockItem, error) {
	query := `
		SELECT stock_item_id, group_id, name, unit_of_measure, opening_qty, opening_valuation, costing_method,
		       COALESCE(hsn_code, ''), COALESCE(gst_rate, 0)
		FROM public.stock_items LIMIT $1 OFFSET $2
	`
	rows, err := db.GetPool().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.StockItem
	for rows.Next() {
		s := &models.StockItem{}
		if err := rows.Scan(&s.StockItemID, &s.GroupID, &s.Name, &s.UnitOfMeasure, &s.OpeningQty, &s.OpeningValuation, &s.CostingMethod, &s.HSNCode, &s.GSTRate); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, nil
}

func GetStockItemsByGroupID(ctx context.Context, groupID string) ([]*models.StockItem, error) {
	query := `
		SELECT stock_item_id, group_id, name, unit_of_measure, opening_qty, opening_valuation, costing_method,
		       COALESCE(hsn_code, ''), COALESCE(gst_rate, 0)
		FROM public.stock_items WHERE group_id = $1
	`
	rows, err := db.GetPool().Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.StockItem
	for rows.Next() {
		s := &models.StockItem{}
		if err := rows.Scan(&s.StockItemID, &s.GroupID, &s.Name, &s.UnitOfMeasure, &s.OpeningQty, &s.OpeningValuation, &s.CostingMethod, &s.HSNCode, &s.GSTRate); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, nil
}

func GetStockItemsByOrganizationID(ctx context.Context, organizationID string) ([]*models.StockItem, error) {
	return GetStockItemsByOrganizationIDs(ctx, organizationID, organizationID)
}

// GetStockItemsByOrganizationIDs returns items whose group belongs to either orgID or companyID
func GetStockItemsByOrganizationIDs(ctx context.Context, orgID, companyID string) ([]*models.StockItem, error) {
	query := `
		SELECT si.stock_item_id, si.group_id, si.name, si.unit_of_measure, si.opening_qty, si.opening_valuation,
		       si.costing_method, COALESCE(si.hsn_code, ''), COALESCE(si.gst_rate, 0)
		FROM public.stock_items si
		LEFT JOIN public.stock_groups sg ON si.group_id = sg.stock_group_id
		WHERE sg.organization_id = $1 OR sg.organization_id = $2 OR si.group_id IS NULL
	`
	rows, err := db.GetPool().Query(ctx, query, orgID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.StockItem
	for rows.Next() {
		s := &models.StockItem{}
		if err := rows.Scan(&s.StockItemID, &s.GroupID, &s.Name, &s.UnitOfMeasure, &s.OpeningQty, &s.OpeningValuation, &s.CostingMethod, &s.HSNCode, &s.GSTRate); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, nil
}

func GetCompanyStockSummary(ctx context.Context, organizationID string) ([]*models.StockSummaryItem, error) {
	queryItems := `
		SELECT stock_item_id, name, unit_of_measure, opening_qty, opening_valuation
		FROM public.stock_items
		WHERE group_id IN (
			SELECT stock_group_id FROM public.stock_groups WHERE organization_id = $1
		)
	`
	rows, err := db.GetPool().Query(ctx, queryItems, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []*models.StockSummaryItem
	summaryMap := make(map[string]*models.StockSummaryItem)

	for rows.Next() {
		var s models.StockSummaryItem
		if err := rows.Scan(&s.StockItemID, &s.Name, &s.UnitOfMeasure, &s.ClosingQty, &s.Valuation); err != nil {
			return nil, err
		}
		summaries = append(summaries, &s)
		summaryMap[s.StockItemID] = &s
	}

	// 3. Fetch all inventory movements for the organization's vouchers
	queryMovements := `
		SELECT ie.stock_item_id, ie.quantity, ie.rate, ie.amount, ie.movement_type
		FROM public.inventory_entries ie
		JOIN public.journal_entries je ON ie.journal_entry_id = je.journal_entry_id
		JOIN public.vouchers v ON je.voucher_id = v.voucher_id
		WHERE v.organization_id = $1
		ORDER BY v.date ASC, ie.inventory_entry_id ASC
	`
	rowsM, err := db.GetPool().Query(ctx, queryMovements, organizationID)
	if err != nil {
		return summaries, nil
	}
	defer rowsM.Close()

	for rowsM.Next() {
		var itemID, moveType string
		var qty, rate, amt float64
		if err := rowsM.Scan(&itemID, &qty, &rate, &amt, &moveType); err != nil {
			return nil, err
		}

		s, exists := summaryMap[itemID]
		if !exists {
			continue
		}

		if moveType == "IN" {
			s.ClosingQty += qty
			s.Valuation += amt
		} else if moveType == "OUT" {
			s.ClosingQty -= qty
			s.Valuation -= amt
		}
	}

	// 4. Calculate average rates and finalize valuations
	for _, s := range summaries {
		if s.ClosingQty > 0 {
			s.AvgRate = s.Valuation / s.ClosingQty
		} else {
			s.ClosingQty = 0
			s.Valuation = 0
			s.AvgRate = 0
		}
	}

	return summaries, nil
}

func GetCompanyOpeningStockValue(ctx context.Context, organizationID string) (float64, error) {
	var total float64
	err := db.GetPool().QueryRow(ctx, `
		SELECT COALESCE(SUM(opening_valuation), 0)
		FROM public.stock_items
		WHERE group_id IN (
			SELECT stock_group_id FROM public.stock_groups WHERE organization_id = $1
		)
	`, organizationID).Scan(&total)
	return total, err
}
