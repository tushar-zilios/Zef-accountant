package routes

import (
	"net/http"

	accountantHandlers "accountant/src/internal/handlers/accountant"
	"accountant/src/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(utils.CORSMiddleware)
	r.Use(conditionalLogger)
	r.Use(handlerLogger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	// Excel template downloads (no auth needed)
	r.Get("/excel/templates/masters", accountantHandlers.DownloadMastersTemplateHandler)
	r.Get("/excel/templates/transactions", accountantHandlers.DownloadTransactionsTemplateHandler)

	r.Route("/organizations", func(subRouter chi.Router) {
		subRouter.Get("/{id}/companies", accountantHandlers.GetCompaniesByOrganizationHandler)
		subRouter.Get("/{id}/account-groups", accountantHandlers.GetAccountGroupsByOrganizationHandler)
		subRouter.Get("/{id}/fiscal-years", accountantHandlers.GetFiscalYearsByOrganizationHandler)
		subRouter.Get("/{id}/stock-groups", accountantHandlers.GetStockGroupsByOrganizationHandler)
		subRouter.Get("/{id}/stock-items", accountantHandlers.GetStockItemsByOrganizationHandler)
		subRouter.Get("/{id}/ledgers", accountantHandlers.GetLedgersByCompanyHandler)
		subRouter.Get("/{id}/vouchers", accountantHandlers.GetVouchersByCompanyHandler)
		subRouter.Get("/{id}/voucher-types", accountantHandlers.GetVoucherTypesByCompanyHandler)
		subRouter.Get("/{id}/next-voucher-number", accountantHandlers.GetNextVoucherNumberHandler)
		subRouter.Get("/{id}/balance-sheet", accountantHandlers.GetOrganizationBalanceSheetHandler)
		subRouter.Get("/{id}/profit-loss", accountantHandlers.GetOrganizationProfitLossHandler)
		subRouter.Get("/{id}/stock-summary", accountantHandlers.GetOrganizationStockSummaryHandler)
		subRouter.Get("/{id}/godowns", accountantHandlers.GetGodownsByOrganizationHandler)
		subRouter.Get("/{id}/godown-stock-summary", accountantHandlers.GetGodownStockSummaryHandler)
		subRouter.Get("/{id}/stock-transfers", accountantHandlers.GetStockTransfersByOrganizationHandler)
		subRouter.Get("/{id}/ratio-analysis", accountantHandlers.GetOrganizationRatioAnalysisHandler)
		subRouter.Get("/{id}/trial-balance", accountantHandlers.GetOrganizationTrialBalanceHandler)
		subRouter.Get("/{id}/gst-report", accountantHandlers.GetOrganizationGSTReportHandler)
		subRouter.Get("/{id}/excel/export/masters", accountantHandlers.ExportMastersHandler)
		subRouter.Get("/{id}/excel/export/transactions", accountantHandlers.ExportTransactionsHandler)
		subRouter.Post("/{id}/excel/import/masters", accountantHandlers.ImportMastersHandler)
		subRouter.Post("/{id}/excel/import/transactions", accountantHandlers.ImportTransactionsHandler)
		subRouter.Post("/{id}/tally/import/masters", accountantHandlers.ImportTallyMastersHandler)
		subRouter.Post("/{id}/tally/import/transactions", accountantHandlers.ImportTallyTransactionsHandler)
	})

	r.Route("/fiscal-years", func(subRouter chi.Router) {
		subRouter.Post("/", accountantHandlers.CreateFiscalYearHandler)
		subRouter.Get("/", accountantHandlers.ListFiscalYearsHandler)
		subRouter.Get("/{id}", accountantHandlers.GetFiscalYearHandler)
		subRouter.Put("/{id}", accountantHandlers.UpdateFiscalYearHandler)
		subRouter.Delete("/{id}", accountantHandlers.DeleteFiscalYearHandler)
		subRouter.Get("/{fiscal_year_id}/einvoices", accountantHandlers.ListEInvoicesByFiscalYearHandler)
		subRouter.Get("/{fiscal_year_id}/eway-bills", accountantHandlers.ListEWayBillsByFiscalYearHandler)
	})

	r.Route("/account-groups", func(subRouter chi.Router) {
		subRouter.Post("/", accountantHandlers.CreateAccountGroupHandler)
		subRouter.Get("/", accountantHandlers.ListAccountGroupsHandler)
		subRouter.Get("/{id}", accountantHandlers.GetAccountGroupHandler)
		subRouter.Put("/{id}", accountantHandlers.UpdateAccountGroupHandler)
		subRouter.Delete("/{id}", accountantHandlers.DeleteAccountGroupHandler)
		subRouter.Get("/{id}/balance", accountantHandlers.GetAccountGroupBalanceHandler)
	})

	r.Route("/voucher-types", func(subRouter chi.Router) {
		subRouter.Post("/", accountantHandlers.CreateVoucherTypeHandler)
		subRouter.Get("/", accountantHandlers.ListVoucherTypesHandler)
		subRouter.Get("/{id}", accountantHandlers.GetVoucherTypeHandler)
		subRouter.Put("/{id}", accountantHandlers.UpdateVoucherTypeHandler)
		subRouter.Delete("/{id}", accountantHandlers.DeleteVoucherTypeHandler)
	})

	r.Route("/stock-groups", func(subRouter chi.Router) {
		subRouter.Post("/", accountantHandlers.CreateStockGroupHandler)
		subRouter.Get("/", accountantHandlers.ListStockGroupsHandler)
		subRouter.Get("/{id}", accountantHandlers.GetStockGroupHandler)
		subRouter.Put("/{id}", accountantHandlers.UpdateStockGroupHandler)
		subRouter.Delete("/{id}", accountantHandlers.DeleteStockGroupHandler)
		subRouter.Get("/{id}/stock-items", accountantHandlers.GetStockItemsByGroupHandler)
	})

	r.Route("/stock-items", func(subRouter chi.Router) {
		subRouter.Post("/", accountantHandlers.CreateStockItemHandler)
		subRouter.Get("/", accountantHandlers.ListStockItemsHandler)
		subRouter.Get("/{id}", accountantHandlers.GetStockItemHandler)
		subRouter.Put("/{id}", accountantHandlers.UpdateStockItemHandler)
		subRouter.Delete("/{id}", accountantHandlers.DeleteStockItemHandler)
	})

	r.Route("/godowns", func(subRouter chi.Router) {
		subRouter.Post("/", accountantHandlers.CreateGodownHandler)
		subRouter.Get("/{id}", accountantHandlers.GetGodownHandler)
		subRouter.Put("/{id}", accountantHandlers.UpdateGodownHandler)
		subRouter.Delete("/{id}", accountantHandlers.DeleteGodownHandler)
	})

	r.Route("/stock-transfers", func(subRouter chi.Router) {
		subRouter.Post("/", accountantHandlers.CreateStockTransferHandler)
		subRouter.Delete("/{id}", accountantHandlers.DeleteStockTransferHandler)
	})

	r.Route("/vouchers", func(subRouter chi.Router) {
		subRouter.Post("/", accountantHandlers.CreateVoucherHandler)
		subRouter.Get("/", accountantHandlers.ListVouchersHandler)
		subRouter.Get("/{id}", accountantHandlers.GetVoucherHandler)
		subRouter.Put("/{id}", accountantHandlers.UpdateVoucherHandler)
		subRouter.Delete("/{id}", accountantHandlers.DeleteVoucherHandler)
		subRouter.Get("/{id}/journal-entries", accountantHandlers.GetJournalEntriesByVoucherHandler)
		subRouter.Post("/{id}/einvoice", accountantHandlers.GenerateEInvoiceHandler)
		subRouter.Get("/{id}/einvoice", accountantHandlers.GetEInvoiceHandler)
		subRouter.Post("/{id}/eway-bill", accountantHandlers.GenerateEWayBillHandler)
		subRouter.Get("/{id}/eway-bill", accountantHandlers.GetEWayBillHandler)
	})

	r.Route("/ledgers", func(subRouter chi.Router) {
		subRouter.Post("/", accountantHandlers.CreateLedgerHandler)
		subRouter.Get("/", accountantHandlers.ListLedgersHandler)
		subRouter.Get("/{id}", accountantHandlers.GetLedgerHandler)
		subRouter.Put("/{id}", accountantHandlers.UpdateLedgerHandler)
		subRouter.Patch("/{id}", accountantHandlers.PatchLedgerHandler)
		subRouter.Delete("/{id}", accountantHandlers.DeleteLedgerHandler)
		subRouter.Get("/{id}/journal-entries", accountantHandlers.GetJournalEntriesByLedgerHandler)
	})

	r.Route("/journal-entries", func(subRouter chi.Router) {
		subRouter.Post("/", accountantHandlers.CreateJournalEntryHandler)
		subRouter.Get("/", accountantHandlers.ListJournalEntriesHandler)
		subRouter.Get("/{id}", accountantHandlers.GetJournalEntryHandler)
		subRouter.Put("/{id}", accountantHandlers.UpdateJournalEntryHandler)
		subRouter.Delete("/{id}", accountantHandlers.DeleteJournalEntryHandler)
		subRouter.Get("/{id}/inventory-entries", accountantHandlers.GetInventoryEntriesByJournalEntryHandler)
	})

	r.Route("/inventory-entries", func(subRouter chi.Router) {
		subRouter.Post("/", accountantHandlers.CreateInventoryEntryHandler)
		subRouter.Get("/", accountantHandlers.ListInventoryEntriesHandler)
		subRouter.Get("/{id}", accountantHandlers.GetInventoryEntryHandler)
		subRouter.Put("/{id}", accountantHandlers.UpdateInventoryEntryHandler)
		subRouter.Delete("/{id}", accountantHandlers.DeleteInventoryEntryHandler)
	})

	return r
}
