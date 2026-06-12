package accountant

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	dbaccountant "accountant/src/internal/db/accountant"
	models "accountant/src/internal/models/accountant"
	"accountant/src/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// generateIRN creates a deterministic 64-char hex IRN from the input fields.
// In production this would call the GST portal API.
func generateIRN(sellerGSTIN, invoiceNo, invoiceDate string) string {
	h := sha256.Sum256([]byte(sellerGSTIN + "|" + invoiceNo + "|" + invoiceDate))
	return fmt.Sprintf("%x", h)
}

func generateAckNo() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%013d", r.Int63n(9000000000000)+1000000000000)
}

func generateEWBNo() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%012d", r.Int63n(900000000000)+100000000000)
}

func GenerateEInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	voucherID := chi.URLParam(r, "id")
	if voucherID == "" {
		utils.WriteError(w, http.StatusBadRequest, "voucher_id is required")
		return
	}

	var req models.EInvoice
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	if req.SellerGSTIN == "" {
		utils.WriteError(w, http.StatusBadRequest, "seller_gstin is required")
		return
	}
	if req.InvoiceNo == "" {
		utils.WriteError(w, http.StatusBadRequest, "invoice_no is required")
		return
	}
	if req.InvoiceDate == "" {
		utils.WriteError(w, http.StatusBadRequest, "invoice_date is required")
		return
	}

	// Derive fiscal_year_id from the parent voucher
	voucher, err := dbaccountant.GetVoucherByID(r.Context(), voucherID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch voucher: "+err.Error())
		return
	}
	if voucher != nil {
		req.FiscalYearID = voucher.FiscalYearID
		req.OrganizationID = voucher.OrganizationID
	}

	req.VoucherID = voucherID
	req.IRN = generateIRN(req.SellerGSTIN, req.InvoiceNo, req.InvoiceDate)
	req.AckNo = generateAckNo()
	req.AckDate = time.Now()
	req.Status = "GENERATED"

	if err := dbaccountant.CreateEInvoice(r.Context(), &req); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to save e-invoice: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, req)
}

func GetEInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	voucherID := chi.URLParam(r, "id")
	if voucherID == "" {
		utils.WriteError(w, http.StatusBadRequest, "voucher_id is required")
		return
	}
	e, err := dbaccountant.GetEInvoiceByVoucherID(r.Context(), voucherID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "e-Invoice not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get e-invoice: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, e)
}

func ListEInvoicesByFiscalYearHandler(w http.ResponseWriter, r *http.Request) {
	fiscalYearID := chi.URLParam(r, "fiscal_year_id")
	orgID := r.URL.Query().Get("organization_id")
	if fiscalYearID == "" {
		utils.WriteError(w, http.StatusBadRequest, "fiscal_year_id is required")
		return
	}
	if orgID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id query param is required")
		return
	}
	list, err := dbaccountant.ListEInvoicesByFiscalYear(r.Context(), fiscalYearID, orgID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to list e-invoices: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.EInvoice{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GenerateEWayBillHandler(w http.ResponseWriter, r *http.Request) {
	voucherID := chi.URLParam(r, "id")
	if voucherID == "" {
		utils.WriteError(w, http.StatusBadRequest, "voucher_id is required")
		return
	}

	var req models.EWayBill
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	if req.SellerGSTIN == "" {
		utils.WriteError(w, http.StatusBadRequest, "seller_gstin is required")
		return
	}

	// Derive fiscal_year_id from the parent voucher
	voucher, err := dbaccountant.GetVoucherByID(r.Context(), voucherID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch voucher: "+err.Error())
		return
	}
	if voucher != nil {
		req.FiscalYearID = voucher.FiscalYearID
		req.OrganizationID = voucher.OrganizationID
	}

	req.VoucherID = voucherID
	req.EWBNo = generateEWBNo()
	req.EWBDate = time.Now()

	// Valid for 1 day per 100 km (min 1 day)
	days := req.Distance / 100
	if days < 1 {
		days = 1
	}
	req.ValidUpto = time.Now().AddDate(0, 0, days).Format("02/01/2006 15:04:00")
	req.Status = "GENERATED"

	if err := dbaccountant.CreateEWayBill(r.Context(), &req); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to save e-way bill: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, req)
}

func GetEWayBillHandler(w http.ResponseWriter, r *http.Request) {
	voucherID := chi.URLParam(r, "id")
	if voucherID == "" {
		utils.WriteError(w, http.StatusBadRequest, "voucher_id is required")
		return
	}
	b, err := dbaccountant.GetEWayBillByVoucherID(r.Context(), voucherID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "e-Way Bill not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get e-way bill: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, b)
}

func ListEWayBillsByFiscalYearHandler(w http.ResponseWriter, r *http.Request) {
	fiscalYearID := chi.URLParam(r, "fiscal_year_id")
	orgID := r.URL.Query().Get("organization_id")
	if fiscalYearID == "" {
		utils.WriteError(w, http.StatusBadRequest, "fiscal_year_id is required")
		return
	}
	if orgID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id query param is required")
		return
	}
	list, err := dbaccountant.ListEWayBillsByFiscalYear(r.Context(), fiscalYearID, orgID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to list e-way bills: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.EWayBill{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}
