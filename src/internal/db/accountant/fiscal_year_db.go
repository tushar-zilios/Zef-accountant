package accountant

import (
	"context"
	"accountant/src/internal/db"
	"accountant/src/internal/models/accountant"
)

// --- FiscalYear Actions ---

func CreateFiscalYear(ctx context.Context, f *models.FiscalYear) error {
	query := `
		INSERT INTO public.fiscal_years (organization_id, year_label, start_date, end_date, is_closed)
		VALUES ($1, $2, $3, $4, COALESCE($5, false))
		RETURNING fiscal_year_id, is_closed
	`
	return db.GetPool().QueryRow(ctx, query, f.OrganizationID, f.YearLabel, f.StartDate, f.EndDate, f.IsClosed).
		Scan(&f.FiscalYearID, &f.IsClosed)
}

func GetFiscalYearByID(ctx context.Context, id string) (*models.FiscalYear, error) {
	query := `
		SELECT fiscal_year_id, organization_id, year_label, start_date, end_date, is_closed
		FROM public.fiscal_years WHERE fiscal_year_id = $1
	`
	f := &models.FiscalYear{}
	err := db.GetPool().QueryRow(ctx, query, id).
		Scan(&f.FiscalYearID, &f.OrganizationID, &f.YearLabel, &f.StartDate, &f.EndDate, &f.IsClosed)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func UpdateFiscalYear(ctx context.Context, f *models.FiscalYear) error {
	query := `
		UPDATE public.fiscal_years
		SET organization_id = $1, year_label = $2, start_date = $3, end_date = $4, is_closed = $5
		WHERE fiscal_year_id = $6
	`
	_, err := db.GetPool().Exec(ctx, query, f.OrganizationID, f.YearLabel, f.StartDate, f.EndDate, f.IsClosed, f.FiscalYearID)
	return err
}

func DeleteFiscalYear(ctx context.Context, id string) error {
	query := `DELETE FROM public.fiscal_years WHERE fiscal_year_id = $1`
	_, err := db.GetPool().Exec(ctx, query, id)
	return err
}

func ListFiscalYears(ctx context.Context, limit, offset int) ([]*models.FiscalYear, error) {
	query := `
		SELECT fiscal_year_id, organization_id, year_label, start_date, end_date, is_closed
		FROM public.fiscal_years LIMIT $1 OFFSET $2
	`
	rows, err := db.GetPool().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.FiscalYear
	for rows.Next() {
		f := &models.FiscalYear{}
		if err := rows.Scan(&f.FiscalYearID, &f.OrganizationID, &f.YearLabel, &f.StartDate, &f.EndDate, &f.IsClosed); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, nil
}

func GetFiscalYearsByOrganizationID(ctx context.Context, organizationID string) ([]*models.FiscalYear, error) {
	query := `
		SELECT fiscal_year_id, organization_id, year_label, start_date, end_date, is_closed
		FROM public.fiscal_years WHERE organization_id = $1
	`
	rows, err := db.GetPool().Query(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.FiscalYear
	for rows.Next() {
		f := &models.FiscalYear{}
		if err := rows.Scan(&f.FiscalYearID, &f.OrganizationID, &f.YearLabel, &f.StartDate, &f.EndDate, &f.IsClosed); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, nil
}
