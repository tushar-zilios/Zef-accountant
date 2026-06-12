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
	utils.WriteJSON(w, http.StatusOK, v)
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
