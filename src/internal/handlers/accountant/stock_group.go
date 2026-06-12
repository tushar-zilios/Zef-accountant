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

// --- StockGroup Handlers ---

func CreateStockGroupHandler(w http.ResponseWriter, r *http.Request) {
	var g models.StockGroup
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
	if err := dbaccountant.CreateStockGroup(r.Context(), &g); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create stock group: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, g)
}

func GetStockGroupHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Stock Group ID is required")
		return
	}
	g, err := dbaccountant.GetStockGroupByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Stock Group not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get stock group: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, g)
}

func UpdateStockGroupHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Stock Group ID is required")
		return
	}
	var g models.StockGroup
	if err := utils.ReadJSON(r, &g); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	g.StockGroupID = id
	if g.OrganizationID == "" {
		utils.WriteError(w, http.StatusBadRequest, "organization_id is required")
		return
	}
	if g.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := dbaccountant.UpdateStockGroup(r.Context(), &g); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to update stock group: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, g)
}

func DeleteStockGroupHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Stock Group ID is required")
		return
	}
	_, err := dbaccountant.GetStockGroupByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Stock Group not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to check stock group: "+err.Error())
		return
	}
	if err := dbaccountant.DeleteStockGroup(r.Context(), id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to delete stock group: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusNoContent, nil)
}

func ListStockGroupsHandler(w http.ResponseWriter, r *http.Request) {
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
	list, err := dbaccountant.ListStockGroups(r.Context(), limit, offset)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to list stock groups: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.StockGroup{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetStockGroupsByCompanyHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}
	list, err := dbaccountant.GetStockGroupsByOrganizationID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get stock groups: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.StockGroup{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetStockGroupsByOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}
	list, err := dbaccountant.GetStockGroupsByOrganizationID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get stock groups: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.StockGroup{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetStockItemsByOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}
	list, err := dbaccountant.GetStockItemsByOrganizationID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get stock items: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.StockItem{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}
