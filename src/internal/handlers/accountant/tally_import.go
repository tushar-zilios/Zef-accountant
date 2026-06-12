package accountant

// Tally XML Import
//
// Tally exports data via Gateway of Tally → Export → XML (Data).
// This handler accepts that XML and maps it to Zef's accountant domain.
//
// Supported Tally XML structures:
//   Masters  : ENVELOPE > BODY > IMPORTDATA > REQUESTDATA > TALLYMESSAGE > GROUP / LEDGER / STOCKITEM
//   Vouchers : ENVELOPE > BODY > IMPORTDATA > REQUESTDATA > TALLYMESSAGE > VOUCHER
//
// Endpoint:
//   POST /organizations/{id}/tally/import/masters      — groups + ledgers + stock items
//   POST /organizations/{id}/tally/import/transactions — vouchers (requires fiscal_year_id + voucher_type_id query params)

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	dbaccountant "accountant/src/internal/db/accountant"
	models "accountant/src/internal/models/accountant"
	"accountant/src/internal/utils"

	"github.com/go-chi/chi/v5"
)

// ---------------------------------------------------------------------------
// Tally XML structs — mirrors the export format produced by Tally ERP 9 / Prime
// ---------------------------------------------------------------------------

type tallyEnvelope struct {
	XMLName xml.Name   `xml:"ENVELOPE"`
	Body    tallyBody  `xml:"BODY"`
}

type tallyBody struct {
	ImportData tallyImportData `xml:"IMPORTDATA"`
}

type tallyImportData struct {
	RequestData tallyRequestData `xml:"REQUESTDATA"`
}

type tallyRequestData struct {
	Messages []tallyMessage `xml:"TALLYMESSAGE"`
}

type tallyMessage struct {
	Groups     []tallyGroup     `xml:"GROUP"`
	Ledgers    []tallyLedger    `xml:"LEDGER"`
	StockItems []tallyStockItem `xml:"STOCKITEM"`
	Vouchers   []tallyVoucher   `xml:"VOUCHER"`
}

// GROUP
type tallyGroup struct {
	Name           string `xml:"NAME,attr"`
	Parent         string `xml:"PARENT"`
	PrimaryGroup   string `xml:"PRIMARYGROUP"`
	IsReserved     string `xml:"ISRESERVED"`  // "Yes"/"No"
	// AFFECTSSTOCK used to detect stock groups vs account groups
	AffectsStock   string `xml:"AFFECTSSTOCK"`
}

// LEDGER
type tallyLedger struct {
	Name               string       `xml:"NAME,attr"`
	Parent             string       `xml:"PARENT"`
	OpeningBalance     string       `xml:"OPENINGBALANCE"` // e.g. "15000.00 Dr" or "-5000.00 Cr"
	Currency           string       `xml:"CURRENCYNAME"`
	TaxType            string       `xml:"TAXTYPE"`
	GSTType            string       `xml:"GSTTYPE"`
	MailingName        string       `xml:"MAILINGNAME"`
	Address            tallyAddress `xml:"ADDRESS"`
	StateName          string       `xml:"STATENAME"`
	CountryName        string       `xml:"COUNTRYNAME"`
	PINCode            string       `xml:"PINCODE"`
	PanITNo            string       `xml:"INCOMETAXNUMBER"`
	TaxIdentifier      string       `xml:"GSTIN"`
	LedgerPhone        string       `xml:"LEDGERPHONE"`
	BankName           string       `xml:"BANKNAME"`
	AccountNo          string       `xml:"BANKACNO"`
	IFSCCode           string       `xml:"IFSCODE"`
	BranchName         string       `xml:"BRANCHNAME"`
	MICRCode           string       `xml:"MICRCODE"`
}

type tallyAddress struct {
	Lines []string `xml:"ADDRESS"`
}

// STOCKITEM
type tallyStockItem struct {
	Name             string `xml:"NAME,attr"`
	Parent           string `xml:"PARENT"`
	UOM              string `xml:"BASEUNITS"`
	OpeningBalance   string `xml:"OPENINGBALANCE"`   // qty
	OpeningRate      string `xml:"OPENINGRATE"`      // rate per unit
	OpeningValue     string `xml:"OPENINGVALUE"`
	HSNCode          string `xml:"HSNCODE"`
	GSTRate          string `xml:"GSTRATE"`
	CostingMethod    string `xml:"COSTINGMETHOD"` // "FIFO"/"LIFO"/"Avg. Cost"
}

// VOUCHER
type tallyVoucher struct {
	Date           string           `xml:"DATE"`
	VoucherNumber  string           `xml:"VOUCHERNUMBER"`
	VoucherType    string           `xml:"VOUCHERTYPENAME"`
	Narration      string           `xml:"NARRATION"`
	Entries        []tallyLedgerEntry `xml:"ALLLEDGERENTRIES.LIST"`
	InventoryEntries []tallyInventoryEntry `xml:"INVENTORYENTRIES.LIST"`
}

type tallyLedgerEntry struct {
	LedgerName    string `xml:"LEDGERNAME"`
	Amount        string `xml:"AMOUNT"`       // negative = Dr in Tally convention
	IsDeemedPositive string `xml:"ISDEBITORNOT"` // alternate field
}

type tallyInventoryEntry struct {
	StockItemName string `xml:"STOCKITEMNAME"`
	Quantity      string `xml:"ACTUALQTY"`
	Rate          string `xml:"RATE"`
	Amount        string `xml:"AMOUNT"`
}

// ---------------------------------------------------------------------------
// ImportTallyMastersHandler
// POST /organizations/{id}/tally/import/masters
// Body: raw Tally XML (Content-Type: application/xml or text/xml)
// ---------------------------------------------------------------------------

func ImportTallyMastersHandler(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	ctx := r.Context()

	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<20)) // 64 MB max
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Failed to read request body: "+err.Error())
		return
	}

	var env tallyEnvelope
	if err := xml.Unmarshal(body, &env); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Tally XML: "+err.Error())
		return
	}

	// Fetch existing account groups by name so we can resolve parent names → IDs
	existingGroups, err := dbaccountant.GetAccountGroupsByOrganizationID(ctx, orgID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch account groups: "+err.Error())
		return
	}
	groupNameToID := make(map[string]string, len(existingGroups))
	for _, g := range existingGroups {
		groupNameToID[strings.ToLower(g.Name)] = g.AccountGroupID
	}

	result := map[string]interface{}{
		"groups_created":      0,
		"ledgers_created":     0,
		"stock_items_created": 0,
		"errors":              []string{},
	}
	var errs []string

	for _, msg := range env.Body.ImportData.RequestData.Messages {
		// -- Groups --
		for _, tg := range msg.Groups {
			name := strings.TrimSpace(tg.Name)
			if name == "" {
				continue
			}
			if _, exists := groupNameToID[strings.ToLower(name)]; exists {
				continue // skip duplicates
			}

			var parentID *string
			if p := strings.TrimSpace(tg.Parent); p != "" {
				if pid, ok := groupNameToID[strings.ToLower(p)]; ok {
					parentID = &pid
				}
			}

			classification := models.AccountClass(tallyGroupClassification(tg.PrimaryGroup, tg.Name))
			isReserved := strings.EqualFold(tg.IsReserved, "Yes")

			g := &models.AccountGroup{
				OrganizationID: orgID,
				ParentID:       parentID,
				Name:           name,
				Classification: classification,
				IsReserved:     &isReserved,
			}
			if err := dbaccountant.CreateAccountGroup(ctx, g); err != nil {
				errs = append(errs, fmt.Sprintf("Group %q: %v", name, err))
			} else {
				groupNameToID[strings.ToLower(name)] = g.AccountGroupID
				result["groups_created"] = result["groups_created"].(int) + 1
			}
		}

		// -- Ledgers --
		for _, tl := range msg.Ledgers {
			name := strings.TrimSpace(tl.Name)
			if name == "" {
				continue
			}

			groupID, ok := groupNameToID[strings.ToLower(strings.TrimSpace(tl.Parent))]
			if !ok {
				errs = append(errs, fmt.Sprintf("Ledger %q: unknown group %q", name, tl.Parent))
				continue
			}

			obAmt, obType := parseTallyAmount(tl.OpeningBalance)
			currency := strings.TrimSpace(tl.Currency)
			if currency == "" {
				currency = "INR"
			}

			l := &models.Ledger{
				OrganizationID:     orgID,
				GroupID:            groupID,
				Name:               name,
				Currency:           currency,
				OpeningBalance:     math.Abs(obAmt),
				OpeningBalanceType: obType,
				MailingName:        strPtrNonEmpty(tl.MailingName),
				Address:            strPtrNonEmpty(strings.Join(tl.Address.Lines, ", ")),
				State:              strPtrNonEmpty(tl.StateName),
				Country:            strPtrNonEmpty(tl.CountryName),
				PinCode:            strPtrNonEmpty(tl.PINCode),
				PanItNo:            strPtrNonEmpty(tl.PanITNo),
				TaxIdentifier:      strPtrNonEmpty(tl.TaxIdentifier),
				BankName:           strPtrNonEmpty(tl.BankName),
				AccountNo:          strPtrNonEmpty(tl.AccountNo),
				IfsCode:            strPtrNonEmpty(tl.IFSCCode),
				Branch:             strPtrNonEmpty(tl.BranchName),
				BsrCode:            strPtrNonEmpty(tl.MICRCode),
				TypeOfReg:          strPtrNonEmpty(tl.GSTType),
			}
			if err := dbaccountant.CreateLedger(ctx, l); err != nil {
				errs = append(errs, fmt.Sprintf("Ledger %q: %v", name, err))
			} else {
				result["ledgers_created"] = result["ledgers_created"].(int) + 1
			}
		}

		// -- Stock Items --
		for _, ts := range msg.StockItems {
			name := strings.TrimSpace(ts.Name)
			if name == "" {
				continue
			}

			uom := strings.TrimSpace(ts.UOM)
			if uom == "" {
				uom = "NOS"
			}

			openingQty, _ := strconv.ParseFloat(cleanTallyNumber(ts.OpeningBalance), 64)
			openingQty = math.Abs(openingQty)

			openingVal, _ := strconv.ParseFloat(cleanTallyNumber(ts.OpeningValue), 64)
			openingVal = math.Abs(openingVal)

			gstRate, _ := strconv.ParseFloat(cleanTallyNumber(ts.GSTRate), 64)

			costingMethod := tallyCostingMethod(ts.CostingMethod)

			var groupID *string
			if gname := strings.TrimSpace(ts.Parent); gname != "" {
				if gid, ok := groupNameToID[strings.ToLower(gname)]; ok {
					groupID = &gid
				}
			}

			s := &models.StockItem{
				GroupID:          groupID,
				Name:             name,
				UnitOfMeasure:    uom,
				OpeningQty:       openingQty,
				OpeningValuation: openingVal,
				CostingMethod:    costingMethod,
				HSNCode:          strings.TrimSpace(ts.HSNCode),
				GSTRate:          gstRate,
			}
			if err := dbaccountant.CreateStockItem(ctx, s); err != nil {
				errs = append(errs, fmt.Sprintf("StockItem %q: %v", name, err))
			} else {
				result["stock_items_created"] = result["stock_items_created"].(int) + 1
			}
		}
	}

	result["errors"] = errs
	utils.WriteJSON(w, http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// ImportTallyTransactionsHandler
// POST /organizations/{id}/tally/import/transactions?fiscal_year_id=X&voucher_type_id=Y
// Body: raw Tally XML
// ---------------------------------------------------------------------------

func ImportTallyTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	ctx := r.Context()

	fiscalYearID := r.URL.Query().Get("fiscal_year_id")
	voucherTypeID := r.URL.Query().Get("voucher_type_id")
	if fiscalYearID == "" || voucherTypeID == "" {
		utils.WriteError(w, http.StatusBadRequest, "fiscal_year_id and voucher_type_id query params are required")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Failed to read request body: "+err.Error())
		return
	}

	var env tallyEnvelope
	if err := xml.Unmarshal(body, &env); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Tally XML: "+err.Error())
		return
	}

	// Build ledger name → ID map for resolving entry ledger names
	existingLedgers, err := dbaccountant.GetLedgersByCompanyID(ctx, orgID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch ledgers: "+err.Error())
		return
	}
	ledgerNameToID := make(map[string]string, len(existingLedgers))
	for _, l := range existingLedgers {
		ledgerNameToID[strings.ToLower(l.Name)] = l.LedgerID
	}

	result := map[string]interface{}{
		"vouchers_created": 0,
		"errors":           []string{},
	}
	var errs []string

	for _, msg := range env.Body.ImportData.RequestData.Messages {
		for _, tv := range msg.Vouchers {
			vNum := strings.TrimSpace(tv.VoucherNumber)
			if vNum == "" {
				continue
			}

			date, err := parseTallyDate(tv.Date)
			if err != nil {
				errs = append(errs, fmt.Sprintf("Voucher %q: invalid date %q", vNum, tv.Date))
				continue
			}

			var entries []*models.JournalEntry
			for seq, te := range tv.Entries {
				ledgerName := strings.TrimSpace(te.LedgerName)
				ledgerID, ok := ledgerNameToID[strings.ToLower(ledgerName)]
				if !ok {
					errs = append(errs, fmt.Sprintf("Voucher %q entry %d: unknown ledger %q", vNum, seq+1, ledgerName))
					continue
				}

				// Tally amount convention: negative = Dr (money going out of the ledger)
				amt, _ := strconv.ParseFloat(cleanTallyNumber(te.Amount), 64)
				var debit, credit float64
				if amt < 0 {
					debit = math.Abs(amt)
				} else {
					credit = amt
				}

				entries = append(entries, &models.JournalEntry{
					LedgerID:      ledgerID,
					Debit:         debit,
					Credit:        credit,
					CurrencyRate:  1,
					SequenceOrder: seq + 1,
				})
			}

			if len(entries) == 0 {
				errs = append(errs, fmt.Sprintf("Voucher %q: no valid journal entries", vNum))
				continue
			}

			narration := strPtrNonEmpty(tv.Narration)
			v := &models.Voucher{
				OrganizationID: orgID,
				FiscalYearID:   fiscalYearID,
				VoucherTypeID:  voucherTypeID,
				VoucherNumber:  vNum,
				Date:           date,
				Narration:      narration,
				Entries:        entries,
			}
			if err := dbaccountant.CreateVoucher(ctx, v); err != nil {
				errs = append(errs, fmt.Sprintf("Voucher %q: %v", vNum, err))
			} else {
				result["vouchers_created"] = result["vouchers_created"].(int) + 1
			}
		}
	}

	result["errors"] = errs
	utils.WriteJSON(w, http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseTallyAmount parses Tally's opening balance format: "15000.00 Dr", "-5000 Cr", "15000 Dr"
// Returns (absolute amount, "DR"/"CR").
func parseTallyAmount(s string) (float64, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, "DR"
	}

	obType := "DR"
	upper := strings.ToUpper(s)
	if strings.HasSuffix(upper, " CR") || strings.HasSuffix(upper, "CR") {
		obType = "CR"
		s = strings.TrimSpace(s[:len(s)-2])
	} else if strings.HasSuffix(upper, " DR") || strings.HasSuffix(upper, "DR") {
		s = strings.TrimSpace(s[:len(s)-2])
	}

	amt, _ := strconv.ParseFloat(cleanTallyNumber(s), 64)
	// Tally sometimes uses negative for Cr balances
	if amt < 0 {
		obType = "CR"
	}
	return math.Abs(amt), obType
}

// parseTallyDate parses Tally date format: "20230401" (YYYYMMDD)
func parseTallyDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if len(s) == 8 {
		return time.Parse("20060102", s)
	}
	// Fallback: try ISO
	return time.Parse("2006-01-02", s)
}

// cleanTallyNumber strips commas and trailing Dr/Cr suffixes from numeric strings
func cleanTallyNumber(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	// Remove units like " Nos", " Kg" — keep only digits, dot, minus
	// Tally puts qty as "10 Nos" or "10.00 Nos"
	parts := strings.Fields(s)
	if len(parts) > 0 {
		if _, err := strconv.ParseFloat(parts[0], 64); err == nil {
			return parts[0]
		}
	}
	return s
}

// tallyGroupClassification maps Tally's primary group names to Zef classification strings
func tallyGroupClassification(primaryGroup, groupName string) string {
	pg := strings.ToLower(strings.TrimSpace(primaryGroup))
	if pg == "" {
		pg = strings.ToLower(strings.TrimSpace(groupName))
	}
	switch {
	case strings.Contains(pg, "capital"), strings.Contains(pg, "reserve"), strings.Contains(pg, "surplus"):
		return "equity"
	case strings.Contains(pg, "loan"), strings.Contains(pg, "liability"), strings.Contains(pg, "current liab"),
		strings.Contains(pg, "provisions"), strings.Contains(pg, "duties"):
		return "liability"
	case strings.Contains(pg, "fixed asset"), strings.Contains(pg, "investments"):
		return "asset"
	case strings.Contains(pg, "current asset"), strings.Contains(pg, "cash"), strings.Contains(pg, "bank"),
		strings.Contains(pg, "sundry debtor"), strings.Contains(pg, "stock"):
		return "asset"
	case strings.Contains(pg, "purchase"), strings.Contains(pg, "direct expense"):
		return "expense"
	case strings.Contains(pg, "sales"), strings.Contains(pg, "direct income"), strings.Contains(pg, "indirect income"):
		return "income"
	case strings.Contains(pg, "indirect expense"), strings.Contains(pg, "misc"):
		return "expense"
	default:
		return "asset"
	}
}

// tallyCostingMethod maps Tally costing method names to Zef values
func tallyCostingMethod(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "lifo":
		return "LIFO"
	case "avg. cost", "average cost", "weighted average":
		return "AVG"
	default:
		return "FIFO"
	}
}

func strPtrNonEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
