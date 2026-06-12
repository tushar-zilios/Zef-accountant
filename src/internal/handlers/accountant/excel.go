package accountant

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	dbaccountant "accountant/src/internal/db/accountant"
	models "accountant/src/internal/models/accountant"
	"accountant/src/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/xuri/excelize/v2"
)

// ---- Export Masters ----

// ExportMastersHandler exports Ledgers, Account Groups, and Stock Items to Excel.
// GET /organizations/{id}/excel/export/masters
func ExportMastersHandler(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	ctx := r.Context()

	f := excelize.NewFile()
	defer f.Close()

	// Remove default sheet
	f.DeleteSheet("Sheet1")

	// -- Ledgers sheet --
	ledgers, err := dbaccountant.GetLedgersByCompanyID(ctx, orgID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch ledgers: "+err.Error())
		return
	}
	lSheet := "Ledgers"
	f.NewSheet(lSheet)
	ledgerHeaders := []string{
		"Name", "Group ID", "Currency", "Opening Balance", "Opening Balance Type",
		"Alias", "Legal Name", "Tax Identifier", "Account No", "IFS Code",
		"Bank Name", "Branch", "BSR Code", "PAN/IT No", "Type of Registration",
		"Mailing Name", "Address", "State", "Country", "Pin Code",
	}
	writeHeaders(f, lSheet, ledgerHeaders)
	for i, l := range ledgers {
		row := i + 2
		setCells(f, lSheet, row, []interface{}{
			l.Name, l.GroupID, l.Currency, l.OpeningBalance, l.OpeningBalanceType,
			strPtrVal(l.Alias), strPtrVal(l.LegalName), strPtrVal(l.TaxIdentifier),
			strPtrVal(l.AccountNo), strPtrVal(l.IfsCode), strPtrVal(l.BankName),
			strPtrVal(l.Branch), strPtrVal(l.BsrCode), strPtrVal(l.PanItNo),
			strPtrVal(l.TypeOfReg), strPtrVal(l.MailingName), strPtrVal(l.Address),
			strPtrVal(l.State), strPtrVal(l.Country), strPtrVal(l.PinCode),
		})
	}

	// -- Account Groups sheet --
	groups, err := dbaccountant.GetAccountGroupsByOrganizationID(ctx, orgID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch account groups: "+err.Error())
		return
	}
	gSheet := "Account Groups"
	f.NewSheet(gSheet)
	groupHeaders := []string{"Name", "Parent ID", "Classification", "Is Reserved"}
	writeHeaders(f, gSheet, groupHeaders)
	for i, g := range groups {
		row := i + 2
		setCells(f, gSheet, row, []interface{}{
			g.Name, strPtrVal(g.ParentID), g.Classification, g.IsReserved,
		})
	}

	// -- Stock Items sheet --
	stockItems, err := dbaccountant.GetStockItemsByOrganizationID(ctx, orgID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch stock items: "+err.Error())
		return
	}
	sSheet := "Stock Items"
	f.NewSheet(sSheet)
	stockHeaders := []string{
		"Name", "Group ID", "Unit of Measure", "Opening Qty", "Opening Valuation",
		"Costing Method", "HSN Code", "GST Rate",
	}
	writeHeaders(f, sSheet, stockHeaders)
	for i, s := range stockItems {
		row := i + 2
		setCells(f, sSheet, row, []interface{}{
			s.Name, strPtrVal(s.GroupID), s.UnitOfMeasure, s.OpeningQty, s.OpeningValuation,
			s.CostingMethod, s.HSNCode, s.GSTRate,
		})
	}

	styleHeaders(f, lSheet, len(ledgerHeaders))
	styleHeaders(f, gSheet, len(groupHeaders))
	styleHeaders(f, sSheet, len(stockHeaders))

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="masters_%s_%s.xlsx"`, orgID[:8], time.Now().Format("20060102")))
	if err := f.Write(w); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to write Excel: "+err.Error())
	}
}

// ---- Export Transactions ----

// ExportTransactionsHandler exports Vouchers and Journal Entries to Excel.
// GET /organizations/{id}/excel/export/transactions
func ExportTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	ctx := r.Context()

	vouchers, err := dbaccountant.GetVouchersByCompanyID(ctx, orgID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch vouchers: "+err.Error())
		return
	}

	f := excelize.NewFile()
	defer f.Close()
	f.DeleteSheet("Sheet1")

	// -- Vouchers sheet --
	vSheet := "Vouchers"
	f.NewSheet(vSheet)
	vHeaders := []string{"Voucher ID", "Voucher Number", "Date", "Voucher Type ID", "Fiscal Year ID", "Narration", "Posted By"}
	writeHeaders(f, vSheet, vHeaders)

	// -- Journal Entries sheet --
	eSheet := "Journal Entries"
	f.NewSheet(eSheet)
	eHeaders := []string{"Voucher ID", "Voucher Number", "Ledger ID", "Debit", "Credit", "Currency Rate", "Narration", "Sequence Order"}
	writeHeaders(f, eSheet, eHeaders)

	eRow := 2
	for i, v := range vouchers {
		vRow := i + 2
		setCells(f, vSheet, vRow, []interface{}{
			v.VoucherID, v.VoucherNumber, v.Date.Format("2006-01-02"),
			v.VoucherTypeID, v.FiscalYearID, strPtrVal(v.Narration), strPtrVal(v.PostedBy),
		})
		for _, e := range v.Entries {
			setCells(f, eSheet, eRow, []interface{}{
				v.VoucherID, v.VoucherNumber, e.LedgerID, e.Debit, e.Credit,
				e.CurrencyRate, strPtrVal(e.Narration), e.SequenceOrder,
			})
			eRow++
		}
	}

	styleHeaders(f, vSheet, len(vHeaders))
	styleHeaders(f, eSheet, len(eHeaders))

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="transactions_%s_%s.xlsx"`, orgID[:8], time.Now().Format("20060102")))
	if err := f.Write(w); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to write Excel: "+err.Error())
	}
}

// ---- Import Masters ----

// ImportMastersHandler imports Ledgers and/or Stock Items from an uploaded Excel file.
// POST /organizations/{id}/excel/import/masters
// Expects multipart/form-data with field "file". Sheets: "Ledgers", "Stock Items".
func ImportMastersHandler(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	ctx := r.Context()

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Failed to parse form: "+err.Error())
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Missing file field: "+err.Error())
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Excel file: "+err.Error())
		return
	}
	defer f.Close()

	result := map[string]interface{}{
		"ledgers_created":     0,
		"stock_items_created": 0,
		"errors":              []string{},
	}
	errs := []string{}

	// -- Import Ledgers --
	if rows, err := f.GetRows("Ledgers"); err == nil && len(rows) > 1 {
		for i, row := range rows[1:] { // skip header
			if len(row) < 5 {
				errs = append(errs, fmt.Sprintf("Ledgers row %d: insufficient columns", i+2))
				continue
			}
			name := strings.TrimSpace(row[0])
			groupID := strings.TrimSpace(row[1])
			if name == "" || groupID == "" {
				errs = append(errs, fmt.Sprintf("Ledgers row %d: name and group_id are required", i+2))
				continue
			}
			currency := strDefault(row, 2, "INR")
			openingBalance, _ := strconv.ParseFloat(strDefault(row, 3, "0"), 64)
			openingBalanceType := strDefault(row, 4, "DR")

			l := &models.Ledger{
				OrganizationID:     orgID,
				GroupID:            groupID,
				Name:               name,
				Currency:           currency,
				OpeningBalance:     openingBalance,
				OpeningBalanceType: openingBalanceType,
				Alias:              strPtrFromRow(row, 5),
				LegalName:          strPtrFromRow(row, 6),
				TaxIdentifier:      strPtrFromRow(row, 7),
				AccountNo:          strPtrFromRow(row, 8),
				IfsCode:            strPtrFromRow(row, 9),
				BankName:           strPtrFromRow(row, 10),
				Branch:             strPtrFromRow(row, 11),
				BsrCode:            strPtrFromRow(row, 12),
				PanItNo:            strPtrFromRow(row, 13),
				TypeOfReg:          strPtrFromRow(row, 14),
				MailingName:        strPtrFromRow(row, 15),
				Address:            strPtrFromRow(row, 16),
				State:              strPtrFromRow(row, 17),
				Country:            strPtrFromRow(row, 18),
				PinCode:            strPtrFromRow(row, 19),
			}
			if err := dbaccountant.CreateLedger(ctx, l); err != nil {
				errs = append(errs, fmt.Sprintf("Ledgers row %d (%s): %v", i+2, name, err))
			} else {
				result["ledgers_created"] = result["ledgers_created"].(int) + 1
			}
		}
	}

	// -- Import Stock Items --
	if rows, err := f.GetRows("Stock Items"); err == nil && len(rows) > 1 {
		for i, row := range rows[1:] {
			if len(row) < 3 {
				errs = append(errs, fmt.Sprintf("Stock Items row %d: insufficient columns", i+2))
				continue
			}
			name := strings.TrimSpace(row[0])
			if name == "" {
				errs = append(errs, fmt.Sprintf("Stock Items row %d: name is required", i+2))
				continue
			}
			groupID := strPtrFromRow(row, 1)
			uom := strDefault(row, 2, "NOS")
			openingQty, _ := strconv.ParseFloat(strDefault(row, 3, "0"), 64)
			openingVal, _ := strconv.ParseFloat(strDefault(row, 4, "0"), 64)
			costingMethod := strDefault(row, 5, "FIFO")
			hsnCode := strDefault(row, 6, "")
			gstRate, _ := strconv.ParseFloat(strDefault(row, 7, "0"), 64)

			s := &models.StockItem{
				GroupID:          groupID,
				Name:             name,
				UnitOfMeasure:    uom,
				OpeningQty:       openingQty,
				OpeningValuation: openingVal,
				CostingMethod:    costingMethod,
				HSNCode:          hsnCode,
				GSTRate:          gstRate,
			}
			if err := dbaccountant.CreateStockItem(ctx, s); err != nil {
				errs = append(errs, fmt.Sprintf("Stock Items row %d (%s): %v", i+2, name, err))
			} else {
				result["stock_items_created"] = result["stock_items_created"].(int) + 1
			}
		}
	}

	result["errors"] = errs
	utils.WriteJSON(w, http.StatusOK, result)
}

// ---- Import Transactions ----

// ImportTransactionsHandler imports Vouchers+JournalEntries from an uploaded Excel file.
// POST /organizations/{id}/excel/import/transactions
// Expects multipart/form-data with "file". Required query params: fiscal_year_id, voucher_type_id.
// Sheet "Vouchers": VoucherNumber, Date, Narration, PostedBy
// Sheet "Journal Entries": VoucherNumber, LedgerID, Debit, Credit, CurrencyRate, Narration, SequenceOrder
func ImportTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	ctx := r.Context()

	fiscalYearID := r.URL.Query().Get("fiscal_year_id")
	voucherTypeID := r.URL.Query().Get("voucher_type_id")
	if fiscalYearID == "" || voucherTypeID == "" {
		utils.WriteError(w, http.StatusBadRequest, "fiscal_year_id and voucher_type_id query params are required")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Failed to parse form: "+err.Error())
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Missing file field: "+err.Error())
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Excel file: "+err.Error())
		return
	}
	defer f.Close()

	result := map[string]interface{}{
		"vouchers_created": 0,
		"errors":           []string{},
	}
	errs := []string{}

	// Parse voucher header rows
	voucherRows, err := f.GetRows("Vouchers")
	if err != nil || len(voucherRows) < 2 {
		utils.WriteError(w, http.StatusBadRequest, "Sheet 'Vouchers' is missing or empty")
		return
	}
	// Parse journal entry rows keyed by voucher number
	entryRows, _ := f.GetRows("Journal Entries")
	entriesByVoucherNum := map[string][]*models.JournalEntry{}
	for i, row := range entryRows[1:] {
		if len(row) < 3 {
			continue
		}
		vNum := strings.TrimSpace(row[0])
		if vNum == "" {
			continue
		}
		ledgerID := strings.TrimSpace(row[1])
		debit, _ := strconv.ParseFloat(strDefault(row, 2, "0"), 64)
		credit, _ := strconv.ParseFloat(strDefault(row, 3, "0"), 64)
		currencyRate, _ := strconv.ParseFloat(strDefault(row, 4, "1"), 64)
		if currencyRate == 0 {
			currencyRate = 1
		}
		narration := strPtrFromRow(row, 5)
		seqOrder, _ := strconv.Atoi(strDefault(row, 6, "0"))
		if seqOrder == 0 {
			seqOrder = i + 1
		}
		entriesByVoucherNum[vNum] = append(entriesByVoucherNum[vNum], &models.JournalEntry{
			LedgerID:      ledgerID,
			Debit:         debit,
			Credit:        credit,
			CurrencyRate:  currencyRate,
			Narration:     narration,
			SequenceOrder: seqOrder,
		})
	}

	for i, row := range voucherRows[1:] {
		if len(row) < 1 {
			continue
		}
		vNum := strings.TrimSpace(row[0])
		if vNum == "" {
			errs = append(errs, fmt.Sprintf("Vouchers row %d: voucher_number is required", i+2))
			continue
		}
		dateStr := strDefault(row, 1, time.Now().Format("2006-01-02"))
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			errs = append(errs, fmt.Sprintf("Vouchers row %d (%s): invalid date '%s'", i+2, vNum, dateStr))
			continue
		}
		narration := strPtrFromRow(row, 2)
		postedBy := strPtrFromRow(row, 3)

		entries, ok := entriesByVoucherNum[vNum]
		if !ok || len(entries) == 0 {
			errs = append(errs, fmt.Sprintf("Vouchers row %d (%s): no journal entries found", i+2, vNum))
			continue
		}

		v := &models.Voucher{
			OrganizationID: orgID,
			FiscalYearID:   fiscalYearID,
			VoucherTypeID:  voucherTypeID,
			VoucherNumber:  vNum,
			Date:           date,
			Narration:      narration,
			PostedBy:       postedBy,
			Entries:        entries,
		}
		if err := dbaccountant.CreateVoucher(ctx, v); err != nil {
			errs = append(errs, fmt.Sprintf("Vouchers row %d (%s): %v", i+2, vNum, err))
		} else {
			result["vouchers_created"] = result["vouchers_created"].(int) + 1
		}
	}

	result["errors"] = errs
	utils.WriteJSON(w, http.StatusOK, result)
}

// ---- Template Download ----

// DownloadMastersTemplateHandler returns a blank Excel template for masters import.
// GET /excel/templates/masters
func DownloadMastersTemplateHandler(w http.ResponseWriter, r *http.Request) {
	f := excelize.NewFile()
	defer f.Close()
	f.DeleteSheet("Sheet1")

	lSheet := "Ledgers"
	f.NewSheet(lSheet)
	writeHeaders(f, lSheet, []string{
		"Name*", "Group ID*", "Currency", "Opening Balance", "Opening Balance Type (DR/CR)*",
		"Alias", "Legal Name", "Tax Identifier", "Account No", "IFS Code",
		"Bank Name", "Branch", "BSR Code", "PAN/IT No", "Type of Registration",
		"Mailing Name", "Address", "State", "Country", "Pin Code",
	})

	sSheet := "Stock Items"
	f.NewSheet(sSheet)
	writeHeaders(f, sSheet, []string{
		"Name*", "Group ID", "Unit of Measure*", "Opening Qty", "Opening Valuation",
		"Costing Method (FIFO/LIFO/AVG)", "HSN Code", "GST Rate (%)",
	})

	styleHeaders(f, lSheet, 20)
	styleHeaders(f, sSheet, 8)

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="masters_import_template.xlsx"`)
	f.Write(w)
}

// DownloadTransactionsTemplateHandler returns a blank Excel template for transactions import.
// GET /excel/templates/transactions
func DownloadTransactionsTemplateHandler(w http.ResponseWriter, r *http.Request) {
	f := excelize.NewFile()
	defer f.Close()
	f.DeleteSheet("Sheet1")

	vSheet := "Vouchers"
	f.NewSheet(vSheet)
	writeHeaders(f, vSheet, []string{
		"Voucher Number*", "Date* (YYYY-MM-DD)", "Narration", "Posted By",
	})

	eSheet := "Journal Entries"
	f.NewSheet(eSheet)
	writeHeaders(f, eSheet, []string{
		"Voucher Number*", "Ledger ID*", "Debit", "Credit", "Currency Rate", "Narration", "Sequence Order",
	})

	styleHeaders(f, vSheet, 4)
	styleHeaders(f, eSheet, 7)

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="transactions_import_template.xlsx"`)
	f.Write(w)
}

// ---- Helpers ----

func writeHeaders(f *excelize.File, sheet string, headers []string) {
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
}

func setCells(f *excelize.File, sheet string, row int, values []interface{}) {
	for col, v := range values {
		cell, _ := excelize.CoordinatesToCellName(col+1, row)
		f.SetCellValue(sheet, cell, v)
	}
}

func styleHeaders(f *excelize.File, sheet string, colCount int) {
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
	})
	if err != nil {
		return
	}
	for col := 1; col <= colCount; col++ {
		cell, _ := excelize.CoordinatesToCellName(col, 1)
		f.SetCellStyle(sheet, cell, cell, style)
	}
}

func strPtrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strPtrFromRow(row []string, idx int) *string {
	if idx >= len(row) {
		return nil
	}
	v := strings.TrimSpace(row[idx])
	if v == "" {
		return nil
	}
	return &v
}

func strDefault(row []string, idx int, def string) string {
	if idx >= len(row) {
		return def
	}
	v := strings.TrimSpace(row[idx])
	if v == "" {
		return def
	}
	return v
}
