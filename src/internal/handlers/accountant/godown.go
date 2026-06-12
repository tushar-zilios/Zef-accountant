package accountant

import (
	"encoding/json"
	"net/http"

	accountantDB "accountant/src/internal/db/accountant"
	models "accountant/src/internal/models/accountant"
	"accountant/src/internal/utils"

	"github.com/go-chi/chi/v5"
)

func CreateGodownHandler(w http.ResponseWriter, r *http.Request) {
	var g models.Godown
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := accountantDB.CreateGodown(r.Context(), &g); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, g)
}

func GetGodownHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	g, err := accountantDB.GetGodownByID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusNotFound, "godown not found")
		return
	}
	utils.WriteJSON(w, http.StatusOK, g)
}

func UpdateGodownHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var g models.Godown
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	g.GodownID = id
	if err := accountantDB.UpdateGodown(r.Context(), &g); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, g)
}

func DeleteGodownHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := accountantDB.DeleteGodown(r.Context(), id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "godown deleted"})
}

func GetGodownsByOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	godowns, err := accountantDB.GetGodownsByOrganizationID(r.Context(), orgID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if godowns == nil {
		godowns = []*models.Godown{}
	}
	utils.WriteJSON(w, http.StatusOK, godowns)
}

func GetGodownStockSummaryHandler(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	summary, err := accountantDB.GetGodownStockSummary(r.Context(), orgID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if summary == nil {
		summary = []*models.GodownStockItem{}
	}
	utils.WriteJSON(w, http.StatusOK, summary)
}

func CreateStockTransferHandler(w http.ResponseWriter, r *http.Request) {
	var t models.StockTransfer
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := accountantDB.CreateStockTransfer(r.Context(), &t); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, t)
}

func GetStockTransfersByOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	transfers, err := accountantDB.GetStockTransfersByOrganizationID(r.Context(), orgID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if transfers == nil {
		transfers = []*models.StockTransfer{}
	}
	utils.WriteJSON(w, http.StatusOK, transfers)
}

func DeleteStockTransferHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := accountantDB.DeleteStockTransfer(r.Context(), id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "transfer deleted"})
}
