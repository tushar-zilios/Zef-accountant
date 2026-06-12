package accountant

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	dbaccountant "accountant/src/internal/db/accountant"
	models "accountant/src/internal/models/accountant"
	"accountant/src/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// --- Ledger Handlers ---

func handleDBError(w http.ResponseWriter, err error, entity string) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			utils.WriteError(w, http.StatusConflict, entity+" already exists: "+pgErr.Detail)
			return true
		case "23503": // foreign_key_violation
			utils.WriteError(w, http.StatusBadRequest, "Invalid reference: "+pgErr.Detail)
			return true
		case "23514": // check_violation
			utils.WriteError(w, http.StatusBadRequest, "Constraint check failed: "+pgErr.Detail)
			return true
		}
	}
	return false
}

func CreateLedgerHandler(w http.ResponseWriter, r *http.Request) {
	var l models.Ledger
	if err := utils.ReadJSON(r, &l); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	if l.OrganizationID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id is required")
		return
	}
	if l.GroupID == "" {
		utils.WriteError(w, http.StatusBadRequest, "group_id is required")
		return
	}
	if l.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if l.OpeningBalanceType == "" {
		utils.WriteError(w, http.StatusBadRequest, "opening_balance_type is required")
		return
	}
	if l.OpeningBalanceType != "DR" && l.OpeningBalanceType != "CR" {
		utils.WriteError(w, http.StatusBadRequest, "opening_balance_type must be either 'DR' or 'CR'")
		return
	}
	if err := dbaccountant.CreateLedger(r.Context(), &l); err != nil {
		if handleDBError(w, err, "Ledger") {
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create ledger: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, l)
}

func GetLedgerHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Ledger ID is required")
		return
	}
	l, err := dbaccountant.GetLedgerByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Ledger not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get ledger: "+err.Error())
		return
	}

	// Generate ETag based on fields
	isActiveVal := true
	if l.IsActive != nil {
		isActiveVal = *l.IsActive
	}
	tagSource := fmt.Sprintf("%s:%s:%s:%s:%s:%f:%s:%t", l.LedgerID, l.OrganizationID, l.GroupID, l.Name, l.Currency, l.OpeningBalance, l.OpeningBalanceType, isActiveVal)
	etag := fmt.Sprintf(`W/"%x"`, sha256.Sum256([]byte(tagSource)))

	// Check If-None-Match header
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
	utils.WriteJSON(w, http.StatusOK, l)
}

func UpdateLedgerHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Ledger ID is required")
		return
	}
	var l models.Ledger
	if err := utils.ReadJSON(r, &l); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	l.LedgerID = id
	if l.OrganizationID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id is required")
		return
	}
	if l.GroupID == "" {
		utils.WriteError(w, http.StatusBadRequest, "group_id is required")
		return
	}
	if l.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if l.OpeningBalanceType == "" {
		utils.WriteError(w, http.StatusBadRequest, "opening_balance_type is required")
		return
	}
	if l.OpeningBalanceType != "DR" && l.OpeningBalanceType != "CR" {
		utils.WriteError(w, http.StatusBadRequest, "opening_balance_type must be either 'DR' or 'CR'")
		return
	}
	if err := dbaccountant.UpdateLedger(r.Context(), &l); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Ledger not found")
			return
		}
		if handleDBError(w, err, "Ledger") {
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to update ledger: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, l)
}

func PatchLedgerHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Ledger ID is required")
		return
	}
	var input models.PatchLedgerInput
	if err := utils.ReadJSON(r, &input); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// Validation
	if input.Name != nil && *input.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "name cannot be empty")
		return
	}
	if input.OrganizationID != nil && *input.OrganizationID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id cannot be empty")
		return
	}
	if input.GroupID != nil && *input.GroupID == "" {
		utils.WriteError(w, http.StatusBadRequest, "group_id cannot be empty")
		return
	}
	if input.OpeningBalanceType != nil {
		typ := *input.OpeningBalanceType
		if typ != "DR" && typ != "CR" {
			utils.WriteError(w, http.StatusBadRequest, "opening_balance_type must be either 'DR' or 'CR'")
			return
		}
	}

	l, err := dbaccountant.PatchLedger(r.Context(), id, &input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Ledger not found")
			return
		}
		if handleDBError(w, err, "Ledger") {
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to patch ledger: "+err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, l)
}

func DeleteLedgerHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Ledger ID is required")
		return
	}
	_, err := dbaccountant.GetLedgerByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Ledger not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to check ledger: "+err.Error())
		return
	}
	if err := dbaccountant.DeleteLedger(r.Context(), id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to delete ledger: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusNoContent, nil)
}

func ListLedgersHandler(w http.ResponseWriter, r *http.Request) {
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
	list, err := dbaccountant.ListLedgers(r.Context(), limit, offset)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to list ledgers: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.Ledger{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetLedgersByCompanyHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}
	list, err := dbaccountant.GetLedgersByCompanyID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get ledgers: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.Ledger{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetCompaniesByOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}
	list, err := dbaccountant.GetCompaniesByOrganizationID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get companies: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.Company{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}
