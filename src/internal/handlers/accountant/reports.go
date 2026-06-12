package accountant

import (
	"net/http"
	"strings"
	"time"

	dbaccountant "accountant/src/internal/db/accountant"
	models "accountant/src/internal/models/accountant"
	"accountant/src/internal/utils"
	"github.com/go-chi/chi/v5"
)

// --- Organization Report Handlers ---

type RatioAnalysisResponse struct {
	CurrentRatio       float64 `json:"current_ratio"`
	QuickRatio         float64 `json:"quick_ratio"`
	DebtToEquityRatio  float64 `json:"debt_to_equity_ratio"`
	GrossProfitMargin  float64 `json:"gross_profit_margin"`
	NetProfitMargin    float64 `json:"net_profit_margin"`
	WorkingCapital     float64 `json:"working_capital"`
	ReturnOnEquity     float64 `json:"roe"`
	ReturnOnAssets     float64 `json:"roa"`
	CurrentAssets      float64 `json:"current_assets"`
	CurrentLiabilities float64 `json:"current_liabilities"`
	StockInHand        float64 `json:"stock_in_hand"`
	LoansLiability     float64 `json:"loans_liability"`
	CapitalAccount     float64 `json:"capital_account"`
	TotalRevenue       float64 `json:"total_revenue"`
	CostOfGoodsSold    float64 `json:"cost_of_goods_sold"`
	NetProfitOrLoss    float64 `json:"net_profit_or_loss"`
	FixedAssets        float64 `json:"fixed_assets"`
	Investments        float64 `json:"investments"`
	TotalDebt          float64 `json:"total_debt"`
	TotalEquity        float64 `json:"total_equity"`
	TotalAssets        float64 `json:"total_assets"`
}

func GetOrganizationBalanceSheetHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}

	balances, err := dbaccountant.GetCompanyAccountGroupBalances(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch balances: "+err.Error())
		return
	}

	var totalIncome, totalExpense float64
	var bsItems []*dbaccountant.GroupBalanceReportItem
	for _, item := range balances {
		switch item.Classification {
		case "INCOME", "REVENUE":
			totalIncome += item.Balance
		case "EXPENSE":
			totalExpense += item.Balance
		default:
			bsItems = append(bsItems, item)
		}
	}

	// Determine whether purchases are tracked through P&L expense accounts.
	// If they are not (i.e. purchases go directly to inventory asset), skip the
	// opening/closing stock adjustment — it only applies to the periodic-inventory
	// P&L model where a Purchases expense account is used.
	var existingPurchaseBalance, existingSalesBalance float64
	for _, item := range balances {
		if item.Classification == "EXPENSE" || item.Classification == "INCOME" {
			if item.Name == "Purchase Accounts" || item.Name == "Direct Expenses" {
				existingPurchaseBalance += item.Balance
			}
			if item.Name == "Sales Accounts" || item.Name == "Direct Incomes" {
				existingSalesBalance += item.Balance
			}
		}
	}

	voucherTotals, err := dbaccountant.GetCompanyVoucherTotals(r.Context(), id)
	var purchaseVoucherExtra float64
	if err == nil {
		if pt, ok := voucherTotals["PURCHASE"]; ok {
			purchaseAmt := pt.TotalDebit
			if purchaseAmt <= 0 {
				purchaseAmt = pt.TotalCredit
			}
			if extra := purchaseAmt - existingPurchaseBalance; extra > 0.009 {
				totalExpense += extra
				purchaseVoucherExtra = extra
			}
		}
		if st, ok := voucherTotals["SALES"]; ok {
			salesAmt := st.TotalCredit
			if salesAmt <= 0 {
				salesAmt = st.TotalDebit
			}
			if extra := salesAmt - existingSalesBalance; extra > 0.009 {
				totalIncome += extra
			}
		}
	}

	// Only apply opening/closing stock P&L adjustment when purchases are tracked
	// as P&L expenses (periodic inventory model). When purchases go directly to
	// inventory asset accounts there is no offsetting purchase expense, so adding
	// closing stock as income would create phantom profit.
	hasPurchaseInPL := existingPurchaseBalance > 0.009 || purchaseVoucherExtra > 0.009
	if hasPurchaseInPL {
		openingStock, err := dbaccountant.GetCompanyOpeningStockValue(r.Context(), id)
		if err == nil {
			stockSummary, err2 := dbaccountant.GetCompanyStockSummary(r.Context(), id)
			var closingStockFromSummary float64
			if err2 == nil {
				for _, s := range stockSummary {
					closingStockFromSummary += s.Valuation
				}
			}

			// When no inventory entries exist (closingStockFromSummary == 0) but purchases
			// have been entered via accounting vouchers, treat the total purchase amount as
			// the closing stock — all purchased goods are assumed to still be in hand.
			// This is the periodic-inventory fallback: stock-in-hand = purchases.
			closingStock := closingStockFromSummary
			totalPurchaseAmt := existingPurchaseBalance + purchaseVoucherExtra
			if closingStock < 0.009 && totalPurchaseAmt > 0.009 {
				closingStock = totalPurchaseAmt
			}

			totalExpense += openingStock
			totalIncome += closingStock

			// If we used a purchase-derived closing stock that is larger than what
			// account_group_db already added to Stock-in-hand (via GetCompanyStockSummary),
			// propagate the difference up through the ASSET hierarchy in bsItems so that
			// the balance sheet asset side matches the equity side.
			extraStockAsset := closingStock - closingStockFromSummary
			if extraStockAsset > 0.009 {
				// Build parent map for upward traversal.
				byID := make(map[string]*dbaccountant.GroupBalanceReportItem, len(bsItems))
				for _, item := range bsItems {
					byID[item.AccountGroupID] = item
				}
				for _, item := range bsItems {
					if strings.EqualFold(item.Name, "Stock-in-hand") {
						cur := item
						for cur != nil {
							cur.Balance += extraStockAsset
							if cur.ParentID == nil {
								break
							}
							cur = byID[*cur.ParentID]
						}
						break
					}
				}
			}
		}
	}

	netProfit := totalIncome - totalExpense
	bsItems = append(bsItems, &dbaccountant.GroupBalanceReportItem{
		AccountGroupID: "synthetic-net-profit",
		Name:           "Net Profit / (Loss)",
		Classification: "EQUITY",
		Balance:        netProfit,
	})

	utils.WriteJSON(w, http.StatusOK, bsItems)
}

func GetOrganizationProfitLossHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}

	balances, err := dbaccountant.GetCompanyAccountGroupBalances(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch balances: "+err.Error())
		return
	}

	var plBalances []*dbaccountant.GroupBalanceReportItem
	for _, item := range balances {
		if item.Classification == "REVENUE" || item.Classification == "INCOME" || item.Classification == "EXPENSE" {
			plBalances = append(plBalances, item)
		}
	}

	var existingPurchaseBalancePL, existingSalesBalancePL float64
	for _, item := range plBalances {
		if item.Name == "Purchase Accounts" || item.Name == "Direct Expenses" {
			existingPurchaseBalancePL += item.Balance
		}
		if item.Name == "Sales Accounts" || item.Name == "Direct Incomes" {
			existingSalesBalancePL += item.Balance
		}
	}

	voucherTotals, err := dbaccountant.GetCompanyVoucherTotals(r.Context(), id)
	var purchaseVoucherExtraPL float64
	if err == nil {
		if pt, ok := voucherTotals["PURCHASE"]; ok {
			purchaseAmt := pt.TotalDebit
			if purchaseAmt <= 0 {
				purchaseAmt = pt.TotalCredit
			}
			extra := purchaseAmt - existingPurchaseBalancePL
			if extra > 0.009 {
				purchaseVoucherExtraPL = extra
				plBalances = append(plBalances, &dbaccountant.GroupBalanceReportItem{
					AccountGroupID: "synthetic-purchases",
					Name:           "Purchases (Vouchers)",
					Classification: "EXPENSE",
					TotalDebit:     extra,
					NetBalanceDR:   extra,
					Balance:        extra,
				})
			}
		}
		if st, ok := voucherTotals["SALES"]; ok {
			salesAmt := st.TotalCredit
			if salesAmt <= 0 {
				salesAmt = st.TotalDebit
			}
			extra := salesAmt - existingSalesBalancePL
			if extra > 0.009 {
				plBalances = append(plBalances, &dbaccountant.GroupBalanceReportItem{
					AccountGroupID: "synthetic-sales",
					Name:           "Sales (Vouchers)",
					Classification: "INCOME",
					TotalCredit:    extra,
					NetBalanceCR:   extra,
					Balance:        extra,
				})
			}
		}
	}

	// Only apply opening/closing stock P&L adjustment when purchases are tracked
	// as P&L expenses. Inventory-routed purchases don't go through a P&L account,
	// so adding closing stock income without a matching purchase expense creates phantom profit.
	if existingPurchaseBalancePL > 0.009 || purchaseVoucherExtraPL > 0.009 {
		openingStock, err := dbaccountant.GetCompanyOpeningStockValue(r.Context(), id)
		if err == nil {
			stockSummary, err := dbaccountant.GetCompanyStockSummary(r.Context(), id)
			var closingStock float64
			if err == nil {
				for _, s := range stockSummary {
					closingStock += s.Valuation
				}
			}
			if openingStock > 0 || closingStock > 0 {
				plBalances = append(plBalances, &dbaccountant.GroupBalanceReportItem{
					AccountGroupID: "synthetic-opening-stock",
					Name:           "Opening Stock",
					Classification: "EXPENSE",
					TotalDebit:     openingStock,
					NetBalanceDR:   openingStock,
					Balance:        openingStock,
				})
				plBalances = append(plBalances, &dbaccountant.GroupBalanceReportItem{
					AccountGroupID: "synthetic-closing-stock",
					Name:           "Closing Stock",
					Classification: "INCOME",
					TotalCredit:    closingStock,
					NetBalanceCR:   closingStock,
					Balance:        closingStock,
				})
			}
		}
	}

	utils.WriteJSON(w, http.StatusOK, plBalances)
}

func GetOrganizationRatioAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}

	balances, err := dbaccountant.GetCompanyAccountGroupBalances(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch balances: "+err.Error())
		return
	}

	findGroupBalance := func(name string) float64 {
		for _, item := range balances {
			if strings.EqualFold(item.Name, name) {
				return item.Balance
			}
		}
		return 0
	}

	currentAssets := findGroupBalance("Current Assets")
	currentLiabilities := findGroupBalance("Current Liabilities")
	stockInHand := findGroupBalance("Stock-in-hand")
	loansLiability := findGroupBalance("Loans (Liability)")
	capitalAccount := findGroupBalance("Capital Account")
	fixedAssets := findGroupBalance("Fixed Assets")
	investments := findGroupBalance("Investments")
	sales := findGroupBalance("Sales Accounts")
	directIncomes := findGroupBalance("Direct Incomes")
	purchases := findGroupBalance("Purchase Accounts")
	directExpenses := findGroupBalance("Direct Expenses")

	var totalIncome, totalExpense float64
	for _, item := range balances {
		switch item.Classification {
		case "INCOME", "REVENUE":
			totalIncome += item.Balance
		case "EXPENSE":
			totalExpense += item.Balance
		}
	}

	var ratioExistingPurchase, ratioExistingSales float64
	for _, item := range balances {
		if item.Classification == "EXPENSE" || item.Classification == "INCOME" {
			if strings.EqualFold(item.Name, "Purchase Accounts") || strings.EqualFold(item.Name, "Direct Expenses") {
				ratioExistingPurchase += item.Balance
			}
			if strings.EqualFold(item.Name, "Sales Accounts") || strings.EqualFold(item.Name, "Direct Incomes") {
				ratioExistingSales += item.Balance
			}
		}
	}

	voucherTotals, err := dbaccountant.GetCompanyVoucherTotals(r.Context(), id)
	var ratioPurchaseExtra float64
	if err == nil {
		if pt, ok := voucherTotals["PURCHASE"]; ok {
			purchaseAmt := pt.TotalDebit
			if purchaseAmt <= 0 {
				purchaseAmt = pt.TotalCredit
			}
			if extra := purchaseAmt - ratioExistingPurchase; extra > 0.009 {
				totalExpense += extra
				ratioPurchaseExtra = extra
			}
		}
		if st, ok := voucherTotals["SALES"]; ok {
			salesAmt := st.TotalCredit
			if salesAmt <= 0 {
				salesAmt = st.TotalDebit
			}
			if extra := salesAmt - ratioExistingSales; extra > 0.009 {
				totalIncome += extra
			}
		}
	}

	if ratioExistingPurchase > 0.009 || ratioPurchaseExtra > 0.009 {
		openingStock, err := dbaccountant.GetCompanyOpeningStockValue(r.Context(), id)
		if err == nil {
			stockSummary, err2 := dbaccountant.GetCompanyStockSummary(r.Context(), id)
			var closingStock float64
			if err2 == nil {
				for _, s := range stockSummary {
					closingStock += s.Valuation
				}
			}
			totalExpense += openingStock
			totalIncome += closingStock
		}
	}

	netProfitOrLoss := totalIncome - totalExpense
	revenue := sales + directIncomes
	costOfGoodsSold := purchases + directExpenses
	grossProfit := revenue - costOfGoodsSold
	workingCapital := currentAssets - currentLiabilities
	totalDebt := currentLiabilities + loansLiability
	totalEquity := capitalAccount + netProfitOrLoss
	totalAssets := currentAssets + fixedAssets + investments

	var currentRatio, quickRatio, debtToEquity, grossProfitMargin, netProfitMargin, roe, roa float64
	if currentLiabilities != 0 {
		currentRatio = currentAssets / currentLiabilities
		quickRatio = (currentAssets - stockInHand) / currentLiabilities
	}
	if totalEquity != 0 {
		debtToEquity = totalDebt / totalEquity
		roe = (netProfitOrLoss / totalEquity) * 100
	}
	if revenue != 0 {
		grossProfitMargin = (grossProfit / revenue) * 100
		netProfitMargin = (netProfitOrLoss / revenue) * 100
	}
	if totalAssets != 0 {
		roa = (netProfitOrLoss / totalAssets) * 100
	}

	resp := RatioAnalysisResponse{
		CurrentRatio:       currentRatio,
		QuickRatio:         quickRatio,
		DebtToEquityRatio:  debtToEquity,
		GrossProfitMargin:  grossProfitMargin,
		NetProfitMargin:    netProfitMargin,
		WorkingCapital:     workingCapital,
		ReturnOnEquity:     roe,
		ReturnOnAssets:     roa,
		CurrentAssets:      currentAssets,
		CurrentLiabilities: currentLiabilities,
		StockInHand:        stockInHand,
		LoansLiability:     loansLiability,
		CapitalAccount:     capitalAccount,
		TotalRevenue:       revenue,
		CostOfGoodsSold:    costOfGoodsSold,
		NetProfitOrLoss:    netProfitOrLoss,
		FixedAssets:        fixedAssets,
		Investments:        investments,
		TotalDebt:          totalDebt,
		TotalEquity:        totalEquity,
		TotalAssets:        totalAssets,
	}
	utils.WriteJSON(w, http.StatusOK, resp)
}

func GetOrganizationStockSummaryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}
	summary, err := dbaccountant.GetCompanyStockSummary(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch stock summary: "+err.Error())
		return
	}
	if summary == nil {
		summary = []*models.StockSummaryItem{}
	}
	utils.WriteJSON(w, http.StatusOK, summary)
}

func GetOrganizationTrialBalanceHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}
	list, err := dbaccountant.GetTrialBalanceLedgers(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get trial balance: "+err.Error())
		return
	}
	if list == nil {
		list = []*dbaccountant.TrialBalanceLedgerItem{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetOrganizationGSTReportHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}

	reportType := r.URL.Query().Get("type")
	if reportType == "" {
		reportType = "GSTR1"
	}
	fiscalYearID := r.URL.Query().Get("fiscal_year_id")

	var from, to string

	if fiscalYearID != "" {
		fy, err := dbaccountant.GetFiscalYearByID(r.Context(), fiscalYearID)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "Invalid fiscal_year_id: "+err.Error())
			return
		}
		from = fy.StartDate.Format("2006-01-02")
		to = fy.EndDate.Format("2006-01-02")
	} else {
		from = r.URL.Query().Get("from")
		to = r.URL.Query().Get("to")
		if from == "" {
			now := time.Now()
			from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")
		}
		if to == "" {
			to = time.Now().Format("2006-01-02")
		}
	}

	if reportType == "GSTR3B" {
		summary, err := dbaccountant.GetGSTR3BSummary(r.Context(), id, from, to)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "Failed to get GSTR3B summary: "+err.Error())
			return
		}
		if summary.Entries == nil {
			summary.Entries = []*models.GSTEntry{}
		}
		utils.WriteJSON(w, http.StatusOK, summary)
		return
	}

	list, err := dbaccountant.GetGSTReportData(r.Context(), id, from, to, reportType)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get GST report: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.GSTEntry{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

