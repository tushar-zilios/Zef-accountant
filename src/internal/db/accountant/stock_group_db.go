package accountant

import (
	"context"
	"accountant/src/internal/db"
	"accountant/src/internal/models/accountant"
)

// --- StockGroup Actions ---

func CreateStockGroup(ctx context.Context, g *models.StockGroup) error {
	query := `
		INSERT INTO public.stock_groups (organization_id, name, parent_id, hsn_code, gst_rate)
		VALUES ($1, $2, $3, COALESCE($4, ''), COALESCE($5, 0))
		RETURNING stock_group_id
	`
	return db.GetPool().QueryRow(ctx, query, g.OrganizationID, g.Name, g.ParentID, g.HSNCode, g.GSTRate).
		Scan(&g.StockGroupID)
}

func GetStockGroupByID(ctx context.Context, id string) (*models.StockGroup, error) {
	query := `
		SELECT stock_group_id, organization_id, name, parent_id,
		       COALESCE(hsn_code, ''), COALESCE(gst_rate, 0)
		FROM public.stock_groups WHERE stock_group_id = $1
	`
	g := &models.StockGroup{}
	err := db.GetPool().QueryRow(ctx, query, id).
		Scan(&g.StockGroupID, &g.OrganizationID, &g.Name, &g.ParentID, &g.HSNCode, &g.GSTRate)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func UpdateStockGroup(ctx context.Context, g *models.StockGroup) error {
	query := `
		UPDATE public.stock_groups
		SET organization_id = $1, name = $2, parent_id = $3, hsn_code = $4, gst_rate = $5
		WHERE stock_group_id = $6
	`
	_, err := db.GetPool().Exec(ctx, query, g.OrganizationID, g.Name, g.ParentID, g.HSNCode, g.GSTRate, g.StockGroupID)
	return err
}

func DeleteStockGroup(ctx context.Context, id string) error {
	query := `DELETE FROM public.stock_groups WHERE stock_group_id = $1`
	_, err := db.GetPool().Exec(ctx, query, id)
	return err
}

func ListStockGroups(ctx context.Context, limit, offset int) ([]*models.StockGroup, error) {
	query := `
		SELECT stock_group_id, organization_id, name, parent_id,
		       COALESCE(hsn_code, ''), COALESCE(gst_rate, 0)
		FROM public.stock_groups LIMIT $1 OFFSET $2
	`
	rows, err := db.GetPool().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.StockGroup
	for rows.Next() {
		g := &models.StockGroup{}
		if err := rows.Scan(&g.StockGroupID, &g.OrganizationID, &g.Name, &g.ParentID, &g.HSNCode, &g.GSTRate); err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, nil
}

func GetStockGroupsByOrganizationID(ctx context.Context, organizationID string) ([]*models.StockGroup, error) {
	query := `
		SELECT stock_group_id, organization_id, name, parent_id,
		       COALESCE(hsn_code, ''), COALESCE(gst_rate, 0)
		FROM public.stock_groups WHERE organization_id = $1
	`
	rows, err := db.GetPool().Query(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.StockGroup
	for rows.Next() {
		g := &models.StockGroup{}
		if err := rows.Scan(&g.StockGroupID, &g.OrganizationID, &g.Name, &g.ParentID, &g.HSNCode, &g.GSTRate); err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, nil
}

// GetStockGroupsByOrganizationIDs returns groups matching either id (handles groups saved under company_id instead of org_id)
func GetStockGroupsByOrganizationIDs(ctx context.Context, orgID, companyID string) ([]*models.StockGroup, error) {
	query := `
		SELECT stock_group_id, organization_id, name, parent_id,
		       COALESCE(hsn_code, ''), COALESCE(gst_rate, 0)
		FROM public.stock_groups WHERE organization_id = $1 OR organization_id = $2
	`
	rows, err := db.GetPool().Query(ctx, query, orgID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.StockGroup
	for rows.Next() {
		g := &models.StockGroup{}
		if err := rows.Scan(&g.StockGroupID, &g.OrganizationID, &g.Name, &g.ParentID, &g.HSNCode, &g.GSTRate); err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, nil
}
