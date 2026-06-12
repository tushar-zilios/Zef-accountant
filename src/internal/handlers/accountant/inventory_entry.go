package accountant

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	dbaccountant "accountant/src/internal/db/accountant"
	"accountant/src/internal/db"
	models "accountant/src/internal/models/accountant"
	"accountant/src/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// --- InventoryEntry Handlers ---

func CreateInventoryEntryHandler(w http.ResponseWriter, r *http.Request) {
	var i models.InventoryEntry
	if err := utils.ReadJSON(r, &i); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	if i.JournalEntryID == "" {
		utils.WriteError(w, http.StatusBadRequest, "journal_entry_id is required")
		return
	}
	if i.StockItemID == "" {
		utils.WriteError(w, http.StatusBadRequest, "stock_item_id is required")
		return
	}
	if i.MovementType != "IN" && i.MovementType != "OUT" {
		utils.WriteError(w, http.StatusBadRequest, "movement_type must be either 'IN' or 'OUT'")
		return
	}
	if err := dbaccountant.CreateInventoryEntry(r.Context(), &i); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create inventory entry: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, i)
}

func GetInventoryEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Inventory Entry ID is required")
		return
	}
	i, err := dbaccountant.GetInventoryEntryByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Inventory Entry not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get inventory entry: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, i)
}

func UpdateInventoryEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Inventory Entry ID is required")
		return
	}
	var i models.InventoryEntry
	if err := utils.ReadJSON(r, &i); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	i.InventoryEntryID = id
	if i.JournalEntryID == "" {
		utils.WriteError(w, http.StatusBadRequest, "journal_entry_id is required")
		return
	}
	if i.StockItemID == "" {
		utils.WriteError(w, http.StatusBadRequest, "stock_item_id is required")
		return
	}
	if i.MovementType != "IN" && i.MovementType != "OUT" {
		utils.WriteError(w, http.StatusBadRequest, "movement_type must be either 'IN' or 'OUT'")
		return
	}
	if err := dbaccountant.UpdateInventoryEntry(r.Context(), &i); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to update inventory entry: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, i)
}

func DeleteInventoryEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Inventory Entry ID is required")
		return
	}
	_, err := dbaccountant.GetInventoryEntryByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Inventory Entry not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to check inventory entry: "+err.Error())
		return
	}
	if err := dbaccountant.DeleteInventoryEntry(r.Context(), id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to delete inventory entry: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusNoContent, nil)
}

func ListInventoryEntriesHandler(w http.ResponseWriter, r *http.Request) {
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
	list, err := dbaccountant.ListInventoryEntries(r.Context(), limit, offset)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to list inventory entries: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.InventoryEntry{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetInventoryEntriesByJournalEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "Journal Entry ID is required")
		return
	}
	list, err := dbaccountant.GetInventoryEntriesByJournalEntryID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get inventory entries: "+err.Error())
		return
	}
	if list == nil {
		list = []*models.InventoryEntry{}
	}
	utils.WriteJSON(w, http.StatusOK, list)
}

func GetNextVoucherNumberHandler(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	if orgID == "" {
		utils.WriteError(w, http.StatusBadRequest, "Organization ID is required")
		return
	}

	voucherTypeName := r.URL.Query().Get("voucher_type_name")
	voucherTypeID := r.URL.Query().Get("voucher_type_id")

	if voucherTypeName == "" && voucherTypeID == "" {
		utils.WriteError(w, http.StatusBadRequest, "Either voucher_type_name or voucher_type_id must be provided")
		return
	}

	var dbVtID, prefix, name string
	var isAutoNumbered bool
	var query string
	var queryArg interface{}

	if voucherTypeID != "" && !strings.HasPrefix(voucherTypeID, "d-") {
		query = "SELECT voucher_type_id, prefix, name, is_auto_numbered FROM public.voucher_types WHERE (organization_id = $1 OR organization_id = '00000000-0000-0000-0000-000000000000') AND voucher_type_id = $2 LIMIT 1"
		queryArg = voucherTypeID
	} else {
		query = "SELECT voucher_type_id, prefix, name, is_auto_numbered FROM public.voucher_types WHERE (organization_id = $1 OR organization_id = '00000000-0000-0000-0000-000000000000') AND LOWER(name) = LOWER($2) LIMIT 1"
		queryArg = voucherTypeName
	}

	err := db.GetPool().QueryRow(r.Context(), query, orgID, queryArg).Scan(&dbVtID, &prefix, &name, &isAutoNumbered)
	if err != nil {
		lname := strings.ToLower(voucherTypeName)
		if voucherTypeID != "" && voucherTypeName == "" {
			if strings.HasPrefix(voucherTypeID, "d-") {
				lname = strings.TrimPrefix(voucherTypeID, "d-")
			}
		}
		
		switch lname {
		case "contra":
			prefix = "CON/"
		case "payment":
			prefix = "PAY/"
		case "receipt":
			prefix = "REC/"
		case "journal":
			prefix = "JRN/"
		case "sales":
			prefix = "SAL/"
		case "purchase":
			prefix = "PUR/"
		default:
			prefix = strings.ToUpper(lname)
			if prefix != "" {
				prefix = prefix + "/"
			} else {
				prefix = "VOU/"
			}
		}
		isAutoNumbered = true
	}

	if !isAutoNumbered {
		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"next_number": prefix,
			"prefix":      prefix,
		})
		return
	}

	var rows pgx.Rows
	if dbVtID != "" {
		rows, err = db.GetPool().Query(r.Context(), "SELECT voucher_number FROM public.vouchers WHERE organization_id = $1 AND voucher_type_id = $2", orgID, dbVtID)
	} else {
		rows, err = db.GetPool().Query(r.Context(), "SELECT voucher_number FROM public.vouchers WHERE organization_id = $1 AND voucher_number LIKE $2", orgID, prefix+"%")
	}

	maxVal := 0
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var vNum string
			if err := rows.Scan(&vNum); err == nil {
				suffix := vNum
				if prefix != "" && strings.HasPrefix(vNum, prefix) {
					suffix = strings.TrimPrefix(vNum, prefix)
				}
				if val, err := strconv.Atoi(suffix); err == nil {
					if val > maxVal {
						maxVal = val
					}
				}
			}
		}
	}

	nextNumber := fmt.Sprintf("%s%d", prefix, maxVal+1)
	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"next_number": nextNumber,
		"prefix":      prefix,
	})
}
