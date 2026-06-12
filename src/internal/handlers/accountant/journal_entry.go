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

// --- JournalEntry Handlers ---

func CreateJournalEntryHandler(w http.ResponseWriter, r *http.Request) {
	var j models.JournalEntry
	if err := utils.ReadJSON(r, &j); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	if j.VoucherID == "" {
		utils.WriteError(w, http.StatusBadRequest, "voucher_id is required")
		return
	}
	if j.LedgerID == "" {
		utils.WriteError(w, http.StatusBadRequest, "ledger_id is required")
		return
	}
	if j.Debit < 0 || j.Credit < 0 {
		utils.WriteError(w, http.StatusBadRequest, "debit and credit must be non-negative")
		return
	}
	// Exclusivity constraint: one must be > 0 and the other must be == 0
	if (j.Debit > 0 && j.Credit != 0) || (j.Credit > 0 && j.Debit != 0) || (j.Debit == 0 && j.Credit == 0) {
		utils.WriteError(w, http.StatusBadRequest, "Exclusivity constraint: either debit must be greater than 0 (and credit equal to 0) or credit must be greater than 0 (and debit equal to 0)")
		return
	}
	if err := dbaccountant.CreateJournalEntry(r.Context(), &j); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create journal entry: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, j)
}

func GetJournalEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Journal Entry ID is required")
		return
	}
	j, err := dbaccountant.GetJournalEntryByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Journal Entry not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get journal entry: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, j)
}

func UpdateJournalEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Journal Entry ID is required")
		return
	}
	var j models.JournalEntry
	if err := utils.ReadJSON(r, &j); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	j.JournalEntryID = id
	if j.VoucherID == "" {
		utils.WriteError(w, http.StatusBadRequest, "voucher_id is required")
		return
	}
	if j.LedgerID == "" {
		utils.WriteError(w, http.StatusBadRequest, "ledger_id is required")
		return
	}
	if j.Debit < 0 || j.Credit < 0 {
		utils.WriteError(w, http.StatusBadRequest, "debit and credit must be non-negative")
		return
	}
	if (j.Debit > 0 && j.Credit != 0) || (j.Credit > 0 && j.Debit != 0) || (j.Debit == 0 && j.Credit == 0) {
		utils.WriteError(w, http.StatusBadRequest, "Exclusivity constraint: either debit must be greater than 0 (and credit equal to 0) or credit must be greater than 0 (and debit equal to 0)")
		return
	}
	if err := dbaccountant.UpdateJournalEntry(r.Context(), &j); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to update journal entry: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, j)
}

func DeleteJournalEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Journal Entry ID is required")
		return
	}
	_, err := dbaccountant.GetJournalEntryByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Journal Entry not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to check journal entry: "+err.Error())
		return
	}
	if err := dbaccountant.DeleteJournalEntry(r.Context(), id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to delete journal entry: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusNoContent, nil)
}

func ListJournalEntriesHandler(w http.ResponseWriter, r *http.Request) {
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
	list, err := dbaccountant.ListJournalEntries(r.Context(), limit, offset)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to list journal entries: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.JournalEntry{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetJournalEntriesByVoucherHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Voucher ID is required")
		return
	}
	list, err := dbaccountant.GetJournalEntriesByVoucherID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get journal entries: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.JournalEntry{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetJournalEntriesByLedgerHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Ledger ID is required")
		return
	}
	list, err := dbaccountant.GetJournalEntriesByLedgerID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get journal entries: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.JournalEntry{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}
