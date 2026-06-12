package accountant

import (
	"errors"
	"net/http"
	"strconv"

	dbaccountant "accountant/src/internal/db/accountant"
	models "accountant/src/internal/models/accountant"
	"accountant/src/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// --- VoucherType Handlers ---

func CreateVoucherTypeHandler(w http.ResponseWriter, r *http.Request) {
	var v models.VoucherType
	if err := utils.ReadJSON(r, &v); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	if v.OrganizationID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id is required")
		return
	}
	if v.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := dbaccountant.CreateVoucherType(r.Context(), &v); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create voucher type: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, v)
}

func GetVoucherTypeHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Voucher Type ID is required")
		return
	}
	v, err := dbaccountant.GetVoucherTypeByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Voucher Type not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get voucher type: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, v)
}

func UpdateVoucherTypeHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Voucher Type ID is required")
		return
	}
	var v models.VoucherType
	if err := utils.ReadJSON(r, &v); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	v.VoucherTypeID = id
	if v.OrganizationID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id is required")
		return
	}
	if v.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := dbaccountant.UpdateVoucherType(r.Context(), &v); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to update voucher type: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, v)
}

func DeleteVoucherTypeHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Voucher Type ID is required")
		return
	}
	vt, err := dbaccountant.GetVoucherTypeByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Voucher Type not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to check voucher type: "+err.Error())
		return
	}
	if (vt.IsReserved != nil && *vt.IsReserved) || (len(id) >= 2 && id[:2] == "d-") {
		utils.WriteError(w, http.StatusForbidden, "System default voucher types cannot be deleted.")
		return
	}
	if err := dbaccountant.DeleteVoucherType(r.Context(), id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to delete voucher type: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusNoContent, nil)
}

func ListVoucherTypesHandler(w http.ResponseWriter, r *http.Request) {
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
	list, err := dbaccountant.ListVoucherTypes(r.Context(), limit, offset)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to list voucher types: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.VoucherType{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetVoucherTypesByCompanyHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}
	includeGlobalStr := r.URL.Query().Get("include_global")
	includeGlobal := includeGlobalStr == "true" || includeGlobalStr == "1"

	list, err := dbaccountant.GetVoucherTypesByOrganizationID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get voucher types: "+err.Error())
		return
	}

	if includeGlobal && id != "00000000-0000-0000-0000-000000000000" {
		globalList, err := dbaccountant.GetVoucherTypesByOrganizationID(r.Context(), "00000000-0000-0000-0000-000000000000")
		if err == nil {
			list = append(list, globalList...)
		}
	}

	if list == nil {
		list = []*models.VoucherType{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}
