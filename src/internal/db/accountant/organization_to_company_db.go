package accountant

import (
	"context"
	"accountant/src/internal/db"
	"accountant/src/internal/models/accountant"
)

// --- OrganizationToCompany Actions ---

func CreateOrganizationToCompany(ctx context.Context, m *models.OrganizationToCompany) error {
	tx, err := db.GetPool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO public.organization_to_company (organization_id, company_id)
		VALUES ($1, $2)
		RETURNING mapping_id
	`
	err = tx.QueryRow(ctx, query, m.OrganizationID, m.CompanyID).Scan(&m.MappingID)
	if err != nil {
		return err
	}

	var hasFY bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.fiscal_years WHERE organization_id = $1)`, m.OrganizationID).Scan(&hasFY)
	if err != nil {
		return err
	}
	if !hasFY {
		if err := CreateDefaultFiscalYear(ctx, tx, m.OrganizationID); err != nil {
			return err
		}
	}

	var hasAG bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.account_groups WHERE organization_id = $1)`, m.OrganizationID).Scan(&hasAG)
	if err != nil {
		return err
	}
	if !hasAG {
		if err := PopulateStandardAccountGroups(ctx, tx, m.OrganizationID); err != nil {
			return err
		}
	}

	var hasVT bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.voucher_types WHERE organization_id = $1)`, m.OrganizationID).Scan(&hasVT)
	if err != nil {
		return err
	}
	if !hasVT {
		if err := PopulateDefaultVoucherTypes(ctx, tx, m.OrganizationID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func GetOrganizationToCompanyByID(ctx context.Context, id string) (*models.OrganizationToCompany, error) {
	query := `
		SELECT mapping_id, organization_id, company_id
		FROM public.organization_to_company WHERE mapping_id = $1
	`
	m := &models.OrganizationToCompany{}
	err := db.GetPool().QueryRow(ctx, query, id).Scan(&m.MappingID, &m.OrganizationID, &m.CompanyID)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func DeleteOrganizationToCompany(ctx context.Context, id string) error {
	query := `DELETE FROM public.organization_to_company WHERE mapping_id = $1`
	_, err := db.GetPool().Exec(ctx, query, id)
	return err
}

func ListOrganizationToCompanies(ctx context.Context, limit, offset int) ([]*models.OrganizationToCompany, error) {
	query := `
		SELECT mapping_id, organization_id, company_id
		FROM public.organization_to_company LIMIT $1 OFFSET $2
	`
	rows, err := db.GetPool().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.OrganizationToCompany
	for rows.Next() {
		m := &models.OrganizationToCompany{}
		if err := rows.Scan(&m.MappingID, &m.OrganizationID, &m.CompanyID); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, nil
}

func GetCompaniesByOrganizationID(ctx context.Context, organizationID string) ([]*models.Company, error) {
	query := `
		SELECT c.company_id, c.name, c.legal_name, c.tax_identifier, c.base_currency, c.created_at, c.updated_at,
			c.alias, c.account_number, c.ifsc_code, c.bank_name, c.branch, c.bsr_code,
			c.pan_number, c.type_of_registration, c.mailing_name, c.address, c.state, c.country, c.pin_code
		FROM public.companies c
		JOIN public.organization_to_company oc ON c.company_id = oc.company_id
		WHERE oc.organization_id = $1
	`
	rows, err := db.GetPool().Query(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Company
	for rows.Next() {
		c := &models.Company{}
		if err := rows.Scan(&c.CompanyID, &c.Name, &c.LegalName, &c.TaxIdentifier, &c.BaseCurrency, &c.CreatedAt, &c.UpdatedAt,
			&c.Alias, &c.AccountNumber, &c.IfscCode, &c.BankName, &c.Branch, &c.BsrCode,
			&c.PanNumber, &c.TypeOfRegistration, &c.MailingName, &c.Address, &c.State, &c.Country, &c.PinCode); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}
