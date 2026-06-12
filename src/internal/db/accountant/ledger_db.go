package accountant

import (
	"context"
	"fmt"

	"accountant/src/internal/db"
	"accountant/src/internal/models/accountant"
)

const ledgerCols = `ledger_id, organization_id, group_id, name, currency, opening_balance, opening_balance_type, is_active,
	legal_name, tax_identifier, base_currency, alias, account_no, ifs_code, bank_name, branch,
	bsr_code, pan_it_no, type_of_reg, mailing_name, address, state, country, pin_code, gst_type`

func scanLedger(row interface {
	Scan(...any) error
}, l *models.Ledger) error {
	return row.Scan(
		&l.LedgerID, &l.OrganizationID, &l.GroupID, &l.Name, &l.Currency,
		&l.OpeningBalance, &l.OpeningBalanceType, &l.IsActive,
		&l.LegalName, &l.TaxIdentifier, &l.BaseCurrency, &l.Alias, &l.AccountNo,
		&l.IfsCode, &l.BankName, &l.Branch, &l.BsrCode, &l.PanItNo, &l.TypeOfReg,
		&l.MailingName, &l.Address, &l.State, &l.Country, &l.PinCode, &l.GSTType,
	)
}

// --- Ledger Actions ---

func CreateLedger(ctx context.Context, l *models.Ledger) error {
	query := `
		INSERT INTO public.ledgers (organization_id, group_id, name, currency, opening_balance, opening_balance_type, is_active,
			legal_name, tax_identifier, base_currency, alias, account_no, ifs_code, bank_name, branch,
			bsr_code, pan_it_no, type_of_reg, mailing_name, address, state, country, pin_code, gst_type)
		VALUES ($1, $2, $3, COALESCE(NULLIF($4, ''), 'INR'), COALESCE($5, 0.0000), $6, COALESCE($7, true),
			$8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, NULLIF($24, ''))
		RETURNING ` + ledgerCols
	return scanLedger(db.GetPool().QueryRow(ctx, query,
		l.OrganizationID, l.GroupID, l.Name, l.Currency, l.OpeningBalance, l.OpeningBalanceType, l.IsActive,
		l.LegalName, l.TaxIdentifier, l.BaseCurrency, l.Alias, l.AccountNo, l.IfsCode, l.BankName, l.Branch,
		l.BsrCode, l.PanItNo, l.TypeOfReg, l.MailingName, l.Address, l.State, l.Country, l.PinCode, l.GSTType,
	), l)
}

func GetLedgerByID(ctx context.Context, id string) (*models.Ledger, error) {
	query := `SELECT ` + ledgerCols + ` FROM public.ledgers WHERE ledger_id = $1`
	l := &models.Ledger{}
	if err := scanLedger(db.GetPool().QueryRow(ctx, query, id), l); err != nil {
		return nil, err
	}
	return l, nil
}

func UpdateLedger(ctx context.Context, l *models.Ledger) error {
	query := `
		UPDATE public.ledgers
		SET organization_id = $1, group_id = $2, name = $3, currency = $4, opening_balance = $5,
		    opening_balance_type = $6, is_active = $7,
		    legal_name = $8, tax_identifier = $9, base_currency = $10, alias = $11, account_no = $12,
		    ifs_code = $13, bank_name = $14, branch = $15, bsr_code = $16, pan_it_no = $17,
		    type_of_reg = $18, mailing_name = $19, address = $20, state = $21, country = $22, pin_code = $23,
		    gst_type = NULLIF($24, '')
		WHERE ledger_id = $25
		RETURNING ` + ledgerCols
	return scanLedger(db.GetPool().QueryRow(ctx, query,
		l.OrganizationID, l.GroupID, l.Name, l.Currency, l.OpeningBalance, l.OpeningBalanceType, l.IsActive,
		l.LegalName, l.TaxIdentifier, l.BaseCurrency, l.Alias, l.AccountNo, l.IfsCode, l.BankName, l.Branch,
		l.BsrCode, l.PanItNo, l.TypeOfReg, l.MailingName, l.Address, l.State, l.Country, l.PinCode,
		l.GSTType, l.LedgerID,
	), l)
}

func PatchLedger(ctx context.Context, id string, input *models.PatchLedgerInput) (*models.Ledger, error) {
	query := "UPDATE public.ledgers SET "
	args := []interface{}{}
	idx := 1

	addField := func(col string, val interface{}) {
		query += fmt.Sprintf("%s = $%d, ", col, idx)
		args = append(args, val)
		idx++
	}

	if input.OrganizationID != nil {
		addField("organization_id", *input.OrganizationID)
	}
	if input.GroupID != nil {
		addField("group_id", *input.GroupID)
	}
	if input.Name != nil {
		addField("name", *input.Name)
	}
	if input.Currency != nil {
		addField("currency", *input.Currency)
	}
	if input.OpeningBalance != nil {
		addField("opening_balance", *input.OpeningBalance)
	}
	if input.OpeningBalanceType != nil {
		addField("opening_balance_type", *input.OpeningBalanceType)
	}
	if input.IsActive != nil {
		addField("is_active", input.IsActive)
	}
	if input.LegalName != nil {
		addField("legal_name", *input.LegalName)
	}
	if input.TaxIdentifier != nil {
		addField("tax_identifier", *input.TaxIdentifier)
	}
	if input.BaseCurrency != nil {
		addField("base_currency", *input.BaseCurrency)
	}
	if input.Alias != nil {
		addField("alias", *input.Alias)
	}
	if input.AccountNo != nil {
		addField("account_no", *input.AccountNo)
	}
	if input.IfsCode != nil {
		addField("ifs_code", *input.IfsCode)
	}
	if input.BankName != nil {
		addField("bank_name", *input.BankName)
	}
	if input.Branch != nil {
		addField("branch", *input.Branch)
	}
	if input.BsrCode != nil {
		addField("bsr_code", *input.BsrCode)
	}
	if input.PanItNo != nil {
		addField("pan_it_no", *input.PanItNo)
	}
	if input.TypeOfReg != nil {
		addField("type_of_reg", *input.TypeOfReg)
	}
	if input.MailingName != nil {
		addField("mailing_name", *input.MailingName)
	}
	if input.Address != nil {
		addField("address", *input.Address)
	}
	if input.State != nil {
		addField("state", *input.State)
	}
	if input.Country != nil {
		addField("country", *input.Country)
	}
	if input.PinCode != nil {
		addField("pin_code", *input.PinCode)
	}
	if input.GSTType != nil {
		addField("gst_type", *input.GSTType)
	}

	if idx == 1 {
		return GetLedgerByID(ctx, id)
	}

	query = query[:len(query)-2]
	query += fmt.Sprintf(" WHERE ledger_id = $%d RETURNING "+ledgerCols, idx)
	args = append(args, id)

	l := &models.Ledger{}
	if err := scanLedger(db.GetPool().QueryRow(ctx, query, args...), l); err != nil {
		return nil, err
	}
	return l, nil
}

func DeleteLedger(ctx context.Context, id string) error {
	_, err := db.GetPool().Exec(ctx, `DELETE FROM public.ledgers WHERE ledger_id = $1`, id)
	return err
}

func ListLedgers(ctx context.Context, limit, offset int) ([]*models.Ledger, error) {
	query := `SELECT ` + ledgerCols + ` FROM public.ledgers LIMIT $1 OFFSET $2`
	rows, err := db.GetPool().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Ledger
	for rows.Next() {
		l := &models.Ledger{}
		if err := scanLedger(rows, l); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, nil
}

type TrialBalanceLedgerItem struct {
	LedgerID     string  `json:"ledger_id"`
	Name         string  `json:"name"`
	GroupID      string  `json:"group_id"`
	GroupName    string  `json:"group_name"`
	OpeningDR    float64 `json:"opening_dr"`
	OpeningCR    float64 `json:"opening_cr"`
	PeriodDebit  float64 `json:"period_debit"`
	PeriodCredit float64 `json:"period_credit"`
	TotalDebit   float64 `json:"total_debit"`
	TotalCredit  float64 `json:"total_credit"`
}

func GetTrialBalanceLedgers(ctx context.Context, organizationID string) ([]*TrialBalanceLedgerItem, error) {
	query := `
		SELECT
			l.ledger_id,
			l.name,
			l.group_id,
			COALESCE(ag.name, '') AS group_name,
			-- Opening balance columns
			CASE WHEN UPPER(l.opening_balance_type) = 'DR' THEN l.opening_balance ELSE 0 END AS opening_dr,
			CASE WHEN UPPER(l.opening_balance_type) = 'CR' THEN l.opening_balance ELSE 0 END AS opening_cr,
			-- Period (transaction) columns
			COALESCE(SUM(je.debit), 0)  AS period_debit,
			COALESCE(SUM(je.credit), 0) AS period_credit
		FROM public.ledgers l
		LEFT JOIN public.account_groups ag ON l.group_id = ag.account_group_id
		LEFT JOIN public.journal_entries je ON l.ledger_id = je.ledger_id
		WHERE l.organization_id = $1
		GROUP BY l.ledger_id, l.name, l.group_id, ag.name, l.opening_balance, l.opening_balance_type
		ORDER BY ag.name, l.name
	`
	rows, err := db.GetPool().Query(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*TrialBalanceLedgerItem
	for rows.Next() {
		item := &TrialBalanceLedgerItem{}
		if err := rows.Scan(&item.LedgerID, &item.Name, &item.GroupID, &item.GroupName,
			&item.OpeningDR, &item.OpeningCR, &item.PeriodDebit, &item.PeriodCredit); err != nil {
			return nil, err
		}
		// Derived totals (opening + period)
		item.TotalDebit = item.OpeningDR + item.PeriodDebit
		item.TotalCredit = item.OpeningCR + item.PeriodCredit
		list = append(list, item)
	}
	return list, nil
}

func GetLedgersByCompanyID(ctx context.Context, organizationID string) ([]*models.Ledger, error) {
	query := `SELECT ` + ledgerCols + ` FROM public.ledgers WHERE organization_id = $1`
	rows, err := db.GetPool().Query(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Ledger
	for rows.Next() {
		l := &models.Ledger{}
		if err := scanLedger(rows, l); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, nil
}
