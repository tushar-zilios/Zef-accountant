package accountant

import (
	"context"
	"fmt"
	"strings"
	"time"
	"accountant/src/internal/db"
	models "accountant/src/internal/models/accountant"
)

// --- Voucher Actions ---

func CreateVoucher(ctx context.Context, v *models.Voucher) error {
	// Validate entry amounts are non-negative and voucher is balanced (total DR = total CR)
	var totalDebit, totalCredit float64
	for _, entry := range v.Entries {
		if entry.Debit < 0 {
			return fmt.Errorf("debit amount cannot be negative")
		}
		if entry.Credit < 0 {
			return fmt.Errorf("credit amount cannot be negative")
		}
		totalDebit += entry.Debit
		totalCredit += entry.Credit
	}
	if len(v.Entries) > 0 && strings.TrimSpace(v.AllowUnbalanced) != "true" {
		diff := totalDebit - totalCredit
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.009 {
			return fmt.Errorf("voucher does not balance: total debit %.2f ≠ total credit %.2f (difference %.2f)", totalDebit, totalCredit, diff)
		}
	}

	tx, err := db.GetPool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO public.vouchers (organization_id, fiscal_year_id, voucher_type_id, voucher_number, date, narration, posted_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, CURRENT_TIMESTAMP))
		RETURNING voucher_id, created_at
	`
	err = tx.QueryRow(ctx, query, v.OrganizationID, v.FiscalYearID, v.VoucherTypeID, v.VoucherNumber, v.Date, v.Narration, v.PostedBy, v.CreatedAt).
		Scan(&v.VoucherID, &v.CreatedAt)
	if err != nil {
		return err
	}

	for _, entry := range v.Entries {
		entryQuery := `
			INSERT INTO public.journal_entries (voucher_id, ledger_id, debit, credit, currency_rate, narration, sequence_order)
			VALUES ($1, $2, COALESCE($3, 0.0000), COALESCE($4, 0.0000), COALESCE($5, 1.000000), $6, $7)
			RETURNING journal_entry_id, debit, credit, currency_rate
		`
		err = tx.QueryRow(ctx, entryQuery, v.VoucherID, entry.LedgerID, entry.Debit, entry.Credit, entry.CurrencyRate, entry.Narration, entry.SequenceOrder).
			Scan(&entry.JournalEntryID, &entry.Debit, &entry.Credit, &entry.CurrencyRate)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func GetVoucherByID(ctx context.Context, id string) (*models.Voucher, error) {
	query := `
		SELECT voucher_id, organization_id, fiscal_year_id, voucher_type_id, voucher_number, date, narration, posted_by, created_at
		FROM public.vouchers WHERE voucher_id = $1
	`
	v := &models.Voucher{}
	err := db.GetPool().QueryRow(ctx, query, id).
		Scan(&v.VoucherID, &v.OrganizationID, &v.FiscalYearID, &v.VoucherTypeID, &v.VoucherNumber, &v.Date, &v.Narration, &v.PostedBy, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func UpdateVoucher(ctx context.Context, v *models.Voucher) error {
	query := `
		UPDATE public.vouchers
		SET organization_id = $1, fiscal_year_id = $2, voucher_type_id = $3, voucher_number = $4, date = $5, narration = $6, posted_by = $7, created_at = $8
		WHERE voucher_id = $9
	`
	_, err := db.GetPool().Exec(ctx, query, v.OrganizationID, v.FiscalYearID, v.VoucherTypeID, v.VoucherNumber, v.Date, v.Narration, v.PostedBy, v.CreatedAt, v.VoucherID)
	return err
}

func DeleteVoucher(ctx context.Context, id string) error {
	query := `DELETE FROM public.vouchers WHERE voucher_id = $1`
	_, err := db.GetPool().Exec(ctx, query, id)
	return err
}

func ListVouchers(ctx context.Context, limit, offset int) ([]*models.Voucher, error) {
	query := `
		SELECT voucher_id, organization_id, fiscal_year_id, voucher_type_id, voucher_number, date, narration, posted_by, created_at
		FROM public.vouchers LIMIT $1 OFFSET $2
	`
	rows, err := db.GetPool().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Voucher
	for rows.Next() {
		v := &models.Voucher{}
		if err := rows.Scan(&v.VoucherID, &v.OrganizationID, &v.FiscalYearID, &v.VoucherTypeID, &v.VoucherNumber, &v.Date, &v.Narration, &v.PostedBy, &v.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, nil
}

func GetVouchersByCompanyID(ctx context.Context, organizationID string) ([]*models.Voucher, error) {
	query := `
		SELECT voucher_id, organization_id, fiscal_year_id, voucher_type_id, voucher_number, date, narration, posted_by, created_at
		FROM public.vouchers WHERE organization_id = $1 ORDER BY date DESC, created_at DESC
	`
	rows, err := db.GetPool().Query(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Voucher
	for rows.Next() {
		v := &models.Voucher{}
		if err := rows.Scan(&v.VoucherID, &v.OrganizationID, &v.FiscalYearID, &v.VoucherTypeID, &v.VoucherNumber, &v.Date, &v.Narration, &v.PostedBy, &v.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, v)
	}

	// Fetch entries for each voucher
	for _, v := range list {
		entryQuery := `
			SELECT journal_entry_id, voucher_id, ledger_id, debit, credit, currency_rate, narration, sequence_order
			FROM public.journal_entries WHERE voucher_id = $1 ORDER BY sequence_order ASC
		`
		eRows, err := db.GetPool().Query(ctx, entryQuery, v.VoucherID)
		if err != nil {
			return nil, err
		}
		v.Entries = []*models.JournalEntry{}
		for eRows.Next() {
			entry := &models.JournalEntry{}
			if err := eRows.Scan(&entry.JournalEntryID, &entry.VoucherID, &entry.LedgerID, &entry.Debit, &entry.Credit, &entry.CurrencyRate, &entry.Narration, &entry.SequenceOrder); err != nil {
				eRows.Close()
				return nil, err
			}
			v.Entries = append(v.Entries, entry)
		}
		eRows.Close()
	}

	return list, nil
}

// VoucherTypeTotals holds aggregated debit/credit for a voucher base type.
type VoucherTypeTotals struct {
	BaseType    string
	TotalDebit  float64
	TotalCredit float64
}

// GetCompanyVoucherTotals returns summed journal entry debits/credits for SALES and PURCHASE
// vouchers for a company. These are used to inject purchase/sales directly into the P&L when
// ledgers may not be properly mapped to Purchase/Sales account groups.
// Only EXPENSE-classified ledger debits are counted for PURCHASE (excludes GST input, stock asset
// debits, etc.) and only INCOME/REVENUE-classified ledger credits are counted for SALES, so that
// balance-sheet-only entries don't pollute the P&L calculation.
func GetCompanyVoucherTotals(ctx context.Context, organizationID string) (map[string]*VoucherTypeTotals, error) {
	query := `
		SELECT
			vt.parent_base_type,
			COALESCE(SUM(CASE
				WHEN vt.parent_base_type = 'PURCHASE' AND ag.classification = 'EXPENSE' THEN je.debit
				ELSE 0
			END), 0) AS total_debit,
			COALESCE(SUM(CASE
				WHEN vt.parent_base_type = 'SALES' AND ag.classification IN ('INCOME', 'REVENUE') THEN je.credit
				ELSE 0
			END), 0) AS total_credit
		FROM public.vouchers v
		JOIN public.voucher_types vt ON v.voucher_type_id = vt.voucher_type_id
		JOIN public.journal_entries je ON je.voucher_id = v.voucher_id
		JOIN public.ledgers l ON je.ledger_id = l.ledger_id
		JOIN public.account_groups ag ON l.group_id = ag.account_group_id
		WHERE v.organization_id = $1
		  AND vt.parent_base_type IN ('SALES', 'PURCHASE')
		  AND (je.debit > 0 OR je.credit > 0)
		GROUP BY vt.parent_base_type
	`
	rows, err := db.GetPool().Query(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*VoucherTypeTotals)
	for rows.Next() {
		t := &VoucherTypeTotals{}
		if err := rows.Scan(&t.BaseType, &t.TotalDebit, &t.TotalCredit); err != nil {
			return nil, err
		}
		result[t.BaseType] = t
	}
	return result, nil
}

// isGSTLedger returns true if the ledger name is a GST tax ledger
func isGSTLedger(name string) bool {
	return strings.Contains(name, "cgst") ||
		strings.Contains(name, "sgst") ||
		strings.Contains(name, "igst") ||
		strings.Contains(name, "utgst") ||
		strings.Contains(name, "gst")
}

// isPartyLedger returns true if this entry represents a settlement/party account
func isPartyLedger(groupName, ledgerName string) bool {
	partyGroups := []string{
		"sundry debtors", "sundry creditors",
		"bank accounts", "cash-in-hand", "bank od accounts",
		"cash in hand", "bank od a/c",
	}
	g := strings.ToLower(groupName)
	for _, pg := range partyGroups {
		if strings.Contains(g, pg) {
			return true
		}
	}
	n := strings.ToLower(ledgerName)
	return strings.Contains(n, "cash") || strings.Contains(n, "bank")
}

// GetGSTReportData calculates and formats GST report entries for an organization in a date range
func GetGSTReportData(ctx context.Context, organizationID string, fromDate, toDate string, reportType string) ([]*models.GSTEntry, error) {
	// GSTR3B fetches both SALES and PURCHASE; GSTR1 = SALES only; GSTR2 = PURCHASE only
	query := `
		SELECT
			v.voucher_id,
			v.voucher_number,
			v.date,
			COALESCE(vt.name, '') AS voucher_type_name,
			COALESCE(vt.parent_base_type, '') AS parent_base_type,
			COALESCE(je.debit, 0) AS debit,
			COALESCE(je.credit, 0) AS credit,
			COALESCE(l.name, '') AS ledger_name,
			COALESCE(ag.name, '') AS group_name,
			COALESCE(l.gst_type, '') AS gst_type
		FROM public.vouchers v
		JOIN public.voucher_types vt ON v.voucher_type_id = vt.voucher_type_id
		LEFT JOIN public.journal_entries je ON v.voucher_id = je.voucher_id
		LEFT JOIN public.ledgers l ON je.ledger_id = l.ledger_id
		LEFT JOIN public.account_groups ag ON l.group_id = ag.account_group_id
		WHERE v.organization_id = $1
		  AND v.date::date >= $2::date AND v.date::date <= $3::date
		  AND (
		       ($4 = 'GSTR1' AND (vt.parent_base_type = 'SALES' OR vt.name ILIKE '%sales%')) OR
		       ($4 = 'GSTR2' AND (vt.parent_base_type = 'PURCHASE' OR vt.name ILIKE '%purchase%')) OR
		       ($4 = 'GSTR3B' AND (
		           vt.parent_base_type IN ('SALES', 'PURCHASE') OR
		           vt.name ILIKE '%sales%' OR vt.name ILIKE '%purchase%'
		       ))
		      )
		ORDER BY v.date DESC, v.created_at DESC, je.sequence_order ASC
	`

	rows, err := db.GetPool().Query(ctx, query, organizationID, fromDate, toDate, reportType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type rawEntry struct {
		debit      float64
		credit     float64
		ledgerName string
		groupName  string
		gstType    string
	}

	type rawVoucher struct {
		voucherNumber string
		date          time.Time
		voucherType   string
		parentBase    string
		entries       []rawEntry
	}

	voucherMap := make(map[string]*rawVoucher)
	var voucherOrder []string

	for rows.Next() {
		var vID, vNum, vTypeName, pBaseType, ledgerName, groupName, gstType string
		var date time.Time
		var debit, credit float64

		if err := rows.Scan(&vID, &vNum, &date, &vTypeName, &pBaseType, &debit, &credit, &ledgerName, &groupName, &gstType); err != nil {
			return nil, err
		}

		v, ok := voucherMap[vID]
		if !ok {
			vName := vTypeName
			if vName == "" {
				vName = pBaseType
			}
			v = &rawVoucher{
				voucherNumber: vNum,
				date:          date,
				voucherType:   vName,
				parentBase:    pBaseType,
				entries:       []rawEntry{},
			}
			voucherMap[vID] = v
			voucherOrder = append(voucherOrder, vID)
		}

		v.entries = append(v.entries, rawEntry{
			debit:      debit,
			credit:     credit,
			ledgerName: ledgerName,
			groupName:  groupName,
			gstType:    gstType,
		})
	}

	var gstEntries []*models.GSTEntry
	for _, id := range voucherOrder {
		v := voucherMap[id]
		var cgst, sgst, igst float64
		var partyAmt float64
		var taxableAmount float64
		partyName := ""

		isSales := v.parentBase == "SALES" ||
			strings.Contains(strings.ToLower(v.voucherType), "sales")

		for _, e := range v.entries {
			name := strings.ToLower(e.ledgerName)
			gt := strings.ToUpper(e.gstType)

			isGST := gt != "" || isGSTLedger(name)
			if isGST {
				// GST component — use whichever side is non-zero
				amt := e.debit
				if amt == 0 {
					amt = e.credit
				}
				// Prefer explicit gst_type; fall back to name matching
				switch {
				case gt == "CGST" || (gt == "" && strings.Contains(name, "cgst")):
					cgst += amt
				case gt == "SGST" || gt == "UTGST" || (gt == "" && (strings.Contains(name, "sgst") || strings.Contains(name, "utgst"))):
					sgst += amt
				case gt == "IGST" || (gt == "" && strings.Contains(name, "igst")):
					igst += amt
				default:
					// gst_type set but not CGST/SGST/IGST/UTGST — treat as generic tax
					igst += amt
				}
			} else if isPartyLedger(e.groupName, e.ledgerName) {
				// Settlement ledger: credit for sales (receivable), debit for purchases (payable)
				amt := e.credit
				if amt == 0 {
					amt = e.debit
				}
				partyAmt += amt
				if partyName == "" {
					partyName = e.ledgerName
				}
			} else {
				// Everything else = taxable supply ledger (stock, purchase a/c, sales a/c, expenses)
				amt := e.debit
				if amt == 0 {
					amt = e.credit
				}
				taxableAmount += amt
			}
		}

		totalTax := cgst + sgst + igst

		// Fallback: if taxable amount wasn't identified from ledgers, derive from party total - tax
		if taxableAmount == 0 && partyAmt > 0 {
			taxableAmount = partyAmt - totalTax
			if taxableAmount < 0 {
				taxableAmount = 0
			}
		}

		if partyName == "" {
			partyName = "—"
		}

		section := "OUTPUT"
		if !isSales {
			section = "INPUT"
		}

		grandTotal := taxableAmount + totalTax

		entry := &models.GSTEntry{
			VoucherNumber: v.voucherNumber,
			Date:          v.date,
			PartyName:     partyName,
			VoucherType:   v.voucherType,
			Section:       section,
			TaxableAmount: taxableAmount,
			CGST:          cgst,
			SGST:          sgst,
			IGST:          igst,
			TotalTax:      totalTax,
			GrandTotal:    grandTotal,
		}

		if totalTax > 0 {
			gstEntries = append(gstEntries, entry)
		}
	}

	return gstEntries, nil
}

// GetGSTR3BSummary builds the monthly return summary from GSTR1 + GSTR2 data
func GetGSTR3BSummary(ctx context.Context, organizationID string, fromDate, toDate string) (*models.GSTR3BSummary, error) {
	entries, err := GetGSTReportData(ctx, organizationID, fromDate, toDate, "GSTR3B")
	if err != nil {
		return nil, err
	}

	summary := &models.GSTR3BSummary{Entries: entries}
	for _, e := range entries {
		if e.Section == "OUTPUT" {
			summary.OutputTaxableAmount += e.TaxableAmount
			summary.OutputCGST += e.CGST
			summary.OutputSGST += e.SGST
			summary.OutputIGST += e.IGST
			summary.OutputTotalTax += e.TotalTax
		} else {
			summary.InputTaxableAmount += e.TaxableAmount
			summary.InputCGST += e.CGST
			summary.InputSGST += e.SGST
			summary.InputIGST += e.IGST
			summary.InputTotalTax += e.TotalTax
		}
	}
	summary.NetCGST = summary.OutputCGST - summary.InputCGST
	summary.NetSGST = summary.OutputSGST - summary.InputSGST
	summary.NetIGST = summary.OutputIGST - summary.InputIGST
	summary.NetTax = summary.NetCGST + summary.NetSGST + summary.NetIGST
	return summary, nil
}

