package accountant

import (
	"context"
	"fmt"
	"time"
	"accountant/src/internal/db"
	"accountant/src/internal/models/accountant"
	"github.com/jackc/pgx/v5"
)

type standardGroup struct {
	Name           string
	Classification string
}

var primaryGroups = []standardGroup{
	{Name: "Capital Account", Classification: "EQUITY"},
	{Name: "Loans (Liability)", Classification: "LIABILITY"},
	{Name: "Current Liabilities", Classification: "LIABILITY"},
	{Name: "Fixed Assets", Classification: "ASSET"},
	{Name: "Investments", Classification: "ASSET"},
	{Name: "Current Assets", Classification: "ASSET"},
	{Name: "Branch / Divisions", Classification: "ASSET"},
	{Name: "Miscellaneous Expenses (Asset)", Classification: "ASSET"},
	{Name: "Sales Accounts", Classification: "INCOME"},
	{Name: "Purchase Accounts", Classification: "EXPENSE"},
	{Name: "Direct Incomes", Classification: "INCOME"},
	{Name: "Indirect Incomes", Classification: "INCOME"},
	{Name: "Direct Expenses", Classification: "EXPENSE"},
	{Name: "Indirect Expenses", Classification: "EXPENSE"},
	{Name: "Suspense A/c", Classification: "ASSET"},
}

type subgroup struct {
	Name           string
	Classification string
	ParentName     string
}

var subgroups = []subgroup{
	{Name: "Reserves & Surplus", Classification: "EQUITY", ParentName: "Capital Account"},
	{Name: "Bank OD A/c", Classification: "LIABILITY", ParentName: "Loans (Liability)"},
	{Name: "Secured Loans", Classification: "LIABILITY", ParentName: "Loans (Liability)"},
	{Name: "Unsecured Loans", Classification: "LIABILITY", ParentName: "Loans (Liability)"},
	{Name: "Duties & Taxes", Classification: "LIABILITY", ParentName: "Current Liabilities"},
	{Name: "Provisions", Classification: "LIABILITY", ParentName: "Current Liabilities"},
	{Name: "Sundry Creditors", Classification: "LIABILITY", ParentName: "Current Liabilities"},
	{Name: "Bank Accounts", Classification: "ASSET", ParentName: "Current Assets"},
	{Name: "Cash-in-hand", Classification: "ASSET", ParentName: "Current Assets"},
	{Name: "Deposits (Asset)", Classification: "ASSET", ParentName: "Current Assets"},
	{Name: "Loans & Advances (Asset)", Classification: "ASSET", ParentName: "Current Assets"},
	{Name: "Stock-in-hand", Classification: "ASSET", ParentName: "Current Assets"},
	{Name: "Sundry Debtors", Classification: "ASSET", ParentName: "Current Assets"},
	// Fixed Assets subgroups (standard Tally norms)
	{Name: "Land & Building", Classification: "ASSET", ParentName: "Fixed Assets"},
	{Name: "Plant & Machinery", Classification: "ASSET", ParentName: "Fixed Assets"},
	{Name: "Furniture & Fixtures", Classification: "ASSET", ParentName: "Fixed Assets"},
	{Name: "Motor Vehicles", Classification: "ASSET", ParentName: "Fixed Assets"},
	{Name: "Office Equipment", Classification: "ASSET", ParentName: "Fixed Assets"},
	// Indirect Expenses subgroups
	{Name: "Salary", Classification: "EXPENSE", ParentName: "Indirect Expenses"},
	{Name: "Rent", Classification: "EXPENSE", ParentName: "Indirect Expenses"},
	{Name: "Bank Charges", Classification: "EXPENSE", ParentName: "Indirect Expenses"},
}

func PopulateStandardAccountGroups(ctx context.Context, tx pgx.Tx, organizationID string) error {
	parentIDs := make(map[string]string)

	// Insert primary groups
	for _, pg := range primaryGroups {
		var id string
		query := `
			INSERT INTO public.account_groups (organization_id, parent_id, name, classification, is_reserved)
			VALUES ($1, NULL, $2, $3, TRUE)
			RETURNING account_group_id
		`
		err := tx.QueryRow(ctx, query, organizationID, pg.Name, pg.Classification).Scan(&id)
		if err != nil {
			return err
		}
		parentIDs[pg.Name] = id
	}

	// Insert subgroups
	for _, sg := range subgroups {
		parentID, exists := parentIDs[sg.ParentName]
		if !exists {
			return fmt.Errorf("parent group %s not found for subgroup %s", sg.ParentName, sg.Name)
		}
		var id string
		query := `
			INSERT INTO public.account_groups (organization_id, parent_id, name, classification, is_reserved)
			VALUES ($1, $2, $3, $4, TRUE)
			RETURNING account_group_id
		`
		err := tx.QueryRow(ctx, query, organizationID, parentID, sg.Name, sg.Classification).Scan(&id)
		if err != nil {
			return err
		}
	}

	return nil
}

// --- Company Actions ---

func CreateCompany(ctx context.Context, c *models.Company) error {
	tx, err := db.GetPool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO public.companies (name, legal_name, tax_identifier, base_currency,
			alias, account_number, ifsc_code, bank_name, branch, bsr_code,
			pan_number, type_of_registration, mailing_name, address, state, country, pin_code)
		VALUES ($1, $2, $3, COALESCE($4, 'INR'),
			$5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17)
		RETURNING company_id, base_currency, created_at, updated_at
	`
	err = tx.QueryRow(ctx, query,
		c.Name, c.LegalName, c.TaxIdentifier, c.BaseCurrency,
		c.Alias, c.AccountNumber, c.IfscCode, c.BankName, c.Branch, c.BsrCode,
		c.PanNumber, c.TypeOfRegistration, c.MailingName, c.Address, c.State, c.Country, c.PinCode).
		Scan(&c.CompanyID, &c.BaseCurrency, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func CreateDefaultFiscalYear(ctx context.Context, tx pgx.Tx, organizationID string) error {
	now := time.Now().UTC()
	year := now.Year()
	var startYear int
	if now.Month() >= time.April {
		startYear = year
	} else {
		startYear = year - 1
	}

	startDate := time.Date(startYear, time.April, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(startYear+1, time.March, 31, 23, 59, 59, 0, time.UTC)
	yearLabel := fmt.Sprintf("FY%d", startYear)

	query := `
		INSERT INTO public.fiscal_years (organization_id, year_label, start_date, end_date, is_closed)
		VALUES ($1, $2, $3, $4, FALSE)
	`
	_, err := tx.Exec(ctx, query, organizationID, yearLabel, startDate, endDate)
	return err
}

func GetCompanyByID(ctx context.Context, id string) (*models.Company, error) {
	query := `
		SELECT company_id, name, legal_name, tax_identifier, base_currency, created_at, updated_at,
			alias, account_number, ifsc_code, bank_name, branch, bsr_code,
			pan_number, type_of_registration, mailing_name, address, state, country, pin_code
		FROM public.companies WHERE company_id = $1
	`
	c := &models.Company{}
	err := db.GetPool().QueryRow(ctx, query, id).
		Scan(&c.CompanyID, &c.Name, &c.LegalName, &c.TaxIdentifier, &c.BaseCurrency, &c.CreatedAt, &c.UpdatedAt,
			&c.Alias, &c.AccountNumber, &c.IfscCode, &c.BankName, &c.Branch, &c.BsrCode,
			&c.PanNumber, &c.TypeOfRegistration, &c.MailingName, &c.Address, &c.State, &c.Country, &c.PinCode)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func UpdateCompany(ctx context.Context, c *models.Company) error {
	query := `
		UPDATE public.companies
		SET name = $1, legal_name = $2, tax_identifier = $3, base_currency = $4,
			alias = $5, account_number = $6, ifsc_code = $7,
			bank_name = $8, branch = $9, bsr_code = $10, pan_number = $11,
			type_of_registration = $12, mailing_name = $13, address = $14,
			state = $15, country = $16, pin_code = $17, updated_at = now()
		WHERE company_id = $18
		RETURNING updated_at
	`
	return db.GetPool().QueryRow(ctx, query,
		c.Name, c.LegalName, c.TaxIdentifier, c.BaseCurrency,
		c.Alias, c.AccountNumber, c.IfscCode,
		c.BankName, c.Branch, c.BsrCode, c.PanNumber,
		c.TypeOfRegistration, c.MailingName, c.Address,
		c.State, c.Country, c.PinCode, c.CompanyID).
		Scan(&c.UpdatedAt)
}

func DeleteCompany(ctx context.Context, id string) error {
	query := `DELETE FROM public.companies WHERE company_id = $1`
	_, err := db.GetPool().Exec(ctx, query, id)
	return err
}

func ForceDeleteCompany(ctx context.Context, id string) error {
	pool := db.GetPool()
	stmts := []string{
		`DELETE FROM public.journal_entries WHERE ledger_id IN (SELECT ledger_id FROM public.ledgers WHERE company_id = $1)`,
		`DELETE FROM public.ledgers WHERE company_id = $1`,
		`DELETE FROM public.vouchers WHERE company_id = $1`,
		`DELETE FROM public.account_groups WHERE company_id = $1`,
		`DELETE FROM public.stock_items WHERE company_id = $1`,
		`DELETE FROM public.stock_groups WHERE company_id = $1`,
		`DELETE FROM public.organization_to_companies WHERE company_id = $1`,
		`DELETE FROM public.companies WHERE company_id = $1`,
	}
	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt, id); err != nil {
			return err
		}
	}
	return nil
}

func ListCompanies(ctx context.Context, limit, offset int) ([]*models.Company, error) {
	query := `
		SELECT company_id, name, legal_name, tax_identifier, base_currency, created_at, updated_at,
			alias, account_number, ifsc_code, bank_name, branch, bsr_code,
			pan_number, type_of_registration, mailing_name, address, state, country, pin_code
		FROM public.companies LIMIT $1 OFFSET $2
	`
	rows, err := db.GetPool().Query(ctx, query, limit, offset)
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
