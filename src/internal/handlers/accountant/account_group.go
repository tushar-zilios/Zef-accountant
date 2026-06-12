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

// --- AccountGroup Handlers ---

func CreateAccountGroupHandler(w http.ResponseWriter, r *http.Request) {
	var g models.AccountGroup
	if err := utils.ReadJSON(r, &g); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	if g.OrganizationID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id is required")
		return
	}
	if g.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if g.Classification == "" {
		utils.WriteError(w, http.StatusBadRequest, "classification is required")
		return
	}
	if err := dbaccountant.CreateAccountGroup(r.Context(), &g); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create account group: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, g)
}

func GetAccountGroupHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Account Group ID is required")
		return
	}
	g, err := dbaccountant.GetAccountGroupByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Account Group not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get account group: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, g)
}

func UpdateAccountGroupHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Account Group ID is required")
		return
	}
	var g models.AccountGroup
	if err := utils.ReadJSON(r, &g); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	g.AccountGroupID = id
	if g.OrganizationID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id is required")
		return
	}
	if g.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if g.Classification == "" {
		utils.WriteError(w, http.StatusBadRequest, "classification is required")
		return
	}
	if err := dbaccountant.UpdateAccountGroup(r.Context(), &g); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to update account group: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, g)
}

func DeleteAccountGroupHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Account Group ID is required")
		return
	}
	_, err := dbaccountant.GetAccountGroupByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Account Group not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to check account group: "+err.Error())
		return
	}
	if err := dbaccountant.DeleteAccountGroup(r.Context(), id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to delete account group: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusNoContent, nil)
}

func GetAccountGroupBalanceHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Account Group ID is required")
		return
	}
	// Check if group exists first
	_, err := dbaccountant.GetAccountGroupByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Account Group not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to check account group: "+err.Error())
		return
	}
	balance, err := dbaccountant.GetAccountGroupBalance(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to fetch account group balance: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, balance)
}

func ListAccountGroupsHandler(w http.ResponseWriter, r *http.Request) {
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
	list, err := dbaccountant.ListAccountGroups(r.Context(), limit, offset)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to list account groups: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.AccountGroup{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetAccountGroupsByCompanyHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}
	list, err := dbaccountant.GetAccountGroupsByOrganizationID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get account groups: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.AccountGroup{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetAccountGroupsByOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}
	list, err := dbaccountant.GetAccountGroupsByOrganizationID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get account groups: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.AccountGroup{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}
