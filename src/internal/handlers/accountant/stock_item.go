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

// --- StockItem Handlers ---

func CreateStockItemHandler(w http.ResponseWriter, r *http.Request) {
	var s models.StockItem
	if err := utils.ReadJSON(r, &s); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	if s.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if s.UnitOfMeasure == "" {
		utils.WriteError(w, http.StatusBadRequest, "unit_of_measure is required")
		return
	}
	if err := dbaccountant.CreateStockItem(r.Context(), &s); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create stock item: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, s)
}

func GetStockItemHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Stock Item ID is required")
		return
	}
	s, err := dbaccountant.GetStockItemByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Stock Item not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get stock item: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, s)
}

func UpdateStockItemHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Stock Item ID is required")
		return
	}
	var s models.StockItem
	if err := utils.ReadJSON(r, &s); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	s.StockItemID = id
	if s.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if s.UnitOfMeasure == "" {
		utils.WriteError(w, http.StatusBadRequest, "unit_of_measure is required")
		return
	}
	if err := dbaccountant.UpdateStockItem(r.Context(), &s); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to update stock item: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, s)
}

func DeleteStockItemHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Stock Item ID is required")
		return
	}
	_, err := dbaccountant.GetStockItemByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Stock Item not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to check stock item: "+err.Error())
		return
	}
	if err := dbaccountant.DeleteStockItem(r.Context(), id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to delete stock item: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusNoContent, nil)
}

func ListStockItemsHandler(w http.ResponseWriter, r *http.Request) {
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
	list, err := dbaccountant.ListStockItems(r.Context(), limit, offset)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to list stock items: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.StockItem{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetStockItemsByGroupHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Group ID is required")
		return
	}
	list, err := dbaccountant.GetStockItemsByGroupID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get stock items: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.StockItem{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetStockItemsByCompanyHandler(w http.ResponseWriter, r *http.Request) {
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
