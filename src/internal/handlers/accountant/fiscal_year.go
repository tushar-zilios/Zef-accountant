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

// --- FiscalYear Handlers ---

func CreateFiscalYearHandler(w http.ResponseWriter, r *http.Request) {
	var f models.FiscalYear
	if err := utils.ReadJSON(r, &f); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	if f.OrganizationID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id is required")
		return
	}
	if f.YearLabel == "" {
		utils.WriteError(w, http.StatusBadRequest, "year_label is required")
		return
	}
	if f.EndDate.Before(f.StartDate) || f.EndDate.Equal(f.StartDate) {
		utils.WriteError(w, http.StatusBadRequest, "end_date must be after start_date")
		return
	}
	// Ensure the organization row exists before the FK-constrained insert.
	if err := dbaccountant.UpsertOrganization(r.Context(), f.OrganizationID); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to provision organization: "+err.Error())
		return
	}
	if err := dbaccountant.CreateFiscalYear(r.Context(), &f); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create fiscal year: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, f)
}

func GetFiscalYearHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Fiscal Year ID is required")
		return
	}
	f, err := dbaccountant.GetFiscalYearByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Fiscal Year not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get fiscal year: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, f)
}

func UpdateFiscalYearHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Fiscal Year ID is required")
		return
	}
	var f models.FiscalYear
	if err := utils.ReadJSON(r, &f); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	f.FiscalYearID = id
	if f.OrganizationID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id is required")
		return
	}
	if f.YearLabel == "" {
		utils.WriteError(w, http.StatusBadRequest, "year_label is required")
		return
	}
	if f.EndDate.Before(f.StartDate) || f.EndDate.Equal(f.StartDate) {
		utils.WriteError(w, http.StatusBadRequest, "end_date must be after start_date")
		return
	}
	if err := dbaccountant.UpdateFiscalYear(r.Context(), &f); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to update fiscal year: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, f)
}

func DeleteFiscalYearHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Fiscal Year ID is required")
		return
	}
	_, err := dbaccountant.GetFiscalYearByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Fiscal Year not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to check fiscal year: "+err.Error())
		return
	}
	if err := dbaccountant.DeleteFiscalYear(r.Context(), id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to delete fiscal year: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusNoContent, nil)
}

func ListFiscalYearsHandler(w http.ResponseWriter, r *http.Request) {
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
	list, err := dbaccountant.ListFiscalYears(r.Context(), limit, offset)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to list fiscal years: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.FiscalYear{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetFiscalYearsByCompanyHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Company ID is required")
		return
	}
	list, err := dbaccountant.GetFiscalYearsByOrganizationID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get fiscal years: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.FiscalYear{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetFiscalYearsByOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}
	list, err := dbaccountant.GetFiscalYearsByOrganizationID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get fiscal years: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.FiscalYear{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}
