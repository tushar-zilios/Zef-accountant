package accountant

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	dbaccountant "accountant/src/internal/db/accountant"
	models "accountant/src/internal/models/accountant"
	"accountant/src/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// --- Voucher Handlers ---

func CreateVoucherHandler(w http.ResponseWriter, r *http.Request) {
	var v models.Voucher
	if err := utils.ReadJSON(r, &v); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	if v.OrganizationID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id is required")
		return
	}
	if v.FiscalYearID == "" {
		utils.WriteError(w, http.StatusBadRequest, "fiscal_year_id is required")
		return
	}
	if v.VoucherTypeID == "" {
		utils.WriteError(w, http.StatusBadRequest, "voucher_type_id is required")
		return
	}
	if v.VoucherNumber == "" {
		utils.WriteError(w, http.StatusBadRequest, "voucher_number is required")
		return
	}
	if v.Date.IsZero() {
		utils.WriteError(w, http.StatusBadRequest, "date is required")
		return
	}
	if err := dbaccountant.CreateVoucher(r.Context(), &v); err != nil {
		if strings.Contains(err.Error(), "does not balance") || strings.Contains(err.Error(), "cannot be negative") {
			utils.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create voucher: "+err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, v)
}

type inventoryEntryDetail struct {
	InventoryEntryID string  `json:"inventory_entry_id"`
	StockItemID      string  `json:"stock_item_id"`
	StockItemName    string  `json:"stock_item_name"`
	Quantity         float64 `json:"quantity"`
	Rate             float64 `json:"rate"`
	Amount           float64 `json:"amount"`
	MovementType     string  `json:"movement_type"`
}

type voucherDetailEntry struct {
	JournalEntryID   string                   `json:"journal_entry_id"`
	LedgerName       string                   `json:"ledger_name"`
	Debit            float64                  `json:"debit"`
	Credit           float64                  `json:"credit"`
	CurrencyRate     float64                  `json:"currency_rate"`
	Narration        *string                  `json:"narration"`
	SequenceOrder    int                      `json:"sequence_order"`
	InventoryEntries []*inventoryEntryDetail  `json:"inventory_entries,omitempty"`
}

type voucherDetailResponse struct {
	VoucherID       string               `json:"voucher_id"`
	VoucherNumber   string               `json:"voucher_number"`
	VoucherTypeName string               `json:"voucher_type_name"`
	Date            string               `json:"date"`
	Narration       *string              `json:"narration"`
	PostedBy        *string              `json:"posted_by"`
	CreatedAt       *string              `json:"created_at"`
	CompanyName     string               `json:"company_name"`
	FiscalYearLabel string               `json:"fiscal_year_label"`
	FiscalYearStart string               `json:"fiscal_year_start"`
	FiscalYearEnd   string               `json:"fiscal_year_end"`
	Entries         []*voucherDetailEntry `json:"entries"`
}

func GetVoucherHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Voucher ID is required")
		return
	}
	v, err := dbaccountant.GetVoucherByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Voucher not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get voucher: "+err.Error())
		return
	}

	resp := &voucherDetailResponse{
		VoucherID:     v.VoucherID,
		VoucherNumber: v.VoucherNumber,
		Date:          v.Date.Format("2006-01-02"),
		Narration:     v.Narration,
		PostedBy:      v.PostedBy,
	}
	if v.CreatedAt != nil {
		s := v.CreatedAt.Format("2006-01-02 15:04:05")
		resp.CreatedAt = &s
	}

	// Voucher type name
	if vt, err := dbaccountant.GetVoucherTypeByID(r.Context(), v.VoucherTypeID); err == nil {
		resp.VoucherTypeName = vt.Name
	}

	// Company name from organization
	if companies, err := dbaccountant.GetCompaniesByOrganizationID(r.Context(), v.OrganizationID); err == nil && len(companies) > 0 {
		c := companies[0]
		if c.Name != nil {
			resp.CompanyName = *c.Name
		}
	}

	// Fiscal year label and dates
	if fy, err := dbaccountant.GetFiscalYearByID(r.Context(), v.FiscalYearID); err == nil {
		resp.FiscalYearLabel = fy.YearLabel
		resp.FiscalYearStart = fy.StartDate.Format("2006-01-02")
		resp.FiscalYearEnd = fy.EndDate.Format("2006-01-02")
	}

	// Journal entries with resolved names and inventory entries
	entries, err := dbaccountant.GetJournalEntriesByVoucherID(r.Context(), id)
	if err == nil {
		resp.Entries = make([]*voucherDetailEntry, 0, len(entries))
		for _, e := range entries {
			de := &voucherDetailEntry{
				JournalEntryID: e.JournalEntryID,
				Debit:          e.Debit,
				Credit:         e.Credit,
				CurrencyRate:   e.CurrencyRate,
				Narration:      e.Narration,
				SequenceOrder:  e.SequenceOrder,
				LedgerName:     e.LedgerID,
			}
			if l, err := dbaccountant.GetLedgerByID(r.Context(), e.LedgerID); err == nil {
				de.LedgerName = l.Name
			}
			// Inventory entries for this journal entry line
			if invs, err := dbaccountant.GetInventoryEntriesByJournalEntryID(r.Context(), e.JournalEntryID); err == nil && len(invs) > 0 {
				de.InventoryEntries = make([]*inventoryEntryDetail, 0, len(invs))
				for _, inv := range invs {
					ied := &inventoryEntryDetail{
						InventoryEntryID: inv.InventoryEntryID,
						StockItemID:      inv.StockItemID,
						StockItemName:    inv.StockItemID,
						Quantity:         inv.Quantity,
						Rate:             inv.Rate,
						Amount:           inv.Amount,
						MovementType:     inv.MovementType,
					}
					if si, err := dbaccountant.GetStockItemByID(r.Context(), inv.StockItemID); err == nil {
						ied.StockItemName = si.Name
					}
					de.InventoryEntries = append(de.InventoryEntries, ied)
				}
			}
			resp.Entries = append(resp.Entries, de)
		}
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

func UpdateVoucherHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Voucher ID is required")
		return
	}
	var v models.Voucher
	if err := utils.ReadJSON(r, &v); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	v.VoucherID = id
	if v.OrganizationID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id is required")
		return
	}
	if v.FiscalYearID == "" {
		utils.WriteError(w, http.StatusBadRequest, "fiscal_year_id is required")
		return
	}
	if v.VoucherTypeID == "" {
		utils.WriteError(w, http.StatusBadRequest, "voucher_type_id is required")
		return
	}
	if v.VoucherNumber == "" {
		utils.WriteError(w, http.StatusBadRequest, "voucher_number is required")
		return
	}
	if v.Date.IsZero() {
		utils.WriteError(w, http.StatusBadRequest, "date is required")
		return
	}
	if err := dbaccountant.UpdateVoucher(r.Context(), &v); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to update voucher: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, v)
}

func DeleteVoucherHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Voucher ID is required")
		return
	}
	_, err := dbaccountant.GetVoucherByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Voucher not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to check voucher: "+err.Error())
		return
	}
	if err := dbaccountant.DeleteVoucher(r.Context(), id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to delete voucher: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusNoContent, nil)
}

func ListVouchersHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	limit := 10
	offset := 0
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			utils.WriteError(w, http.StatusBadRequest, "Invalid limit parameter")
			return
		}
	}
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			utils.WriteError(w, http.StatusBadRequest, "Invalid offset parameter")
			return
		}
	}
	list, err := dbaccountant.ListVouchers(r.Context(), limit, offset)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to list vouchers: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.Voucher{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetVouchersByCompanyHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}
	list, err := dbaccountant.GetVouchersByCompanyID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get vouchers: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.Voucher{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}
