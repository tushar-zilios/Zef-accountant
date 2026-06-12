package accountant

import (
	"context"
	"fmt"
	"accountant/src/internal/db"
	"accountant/src/internal/models/accountant"
	"github.com/jackc/pgx/v5"
)

// --- VoucherType Actions ---

func CreateVoucherType(ctx context.Context, v *models.VoucherType) error {
	query := `
		INSERT INTO public.voucher_types (organization_id, name, prefix, is_auto_numbered, is_reserved, parent_base_type)
		VALUES ($1, $2, $3, COALESCE($4, true), COALESCE($5, false), $6)
		RETURNING voucher_type_id, is_auto_numbered, is_reserved, parent_base_type
	`
	return db.GetPool().QueryRow(ctx, query, v.OrganizationID, v.Name, v.Prefix, v.IsAutoNumbered, v.IsReserved, v.ParentBaseType).
		Scan(&v.VoucherTypeID, &v.IsAutoNumbered, &v.IsReserved, &v.ParentBaseType)
}

func GetVoucherTypeByID(ctx context.Context, id string) (*models.VoucherType, error) {
	query := `
		SELECT voucher_type_id, organization_id, name, prefix, is_auto_numbered, is_reserved, parent_base_type
		FROM public.voucher_types WHERE voucher_type_id = $1
	`
	v := &models.VoucherType{}
	err := db.GetPool().QueryRow(ctx, query, id).
		Scan(&v.VoucherTypeID, &v.OrganizationID, &v.Name, &v.Prefix, &v.IsAutoNumbered, &v.IsReserved, &v.ParentBaseType)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func UpdateVoucherType(ctx context.Context, v *models.VoucherType) error {
	query := `
		UPDATE public.voucher_types
		SET organization_id = $1, name = $2, prefix = $3, is_auto_numbered = $4, is_reserved = $5, parent_base_type = $6
		WHERE voucher_type_id = $7
	`
	_, err := db.GetPool().Exec(ctx, query, v.OrganizationID, v.Name, v.Prefix, v.IsAutoNumbered, v.IsReserved, v.ParentBaseType, v.VoucherTypeID)
	return err
}

func DeleteVoucherType(ctx context.Context, id string) error {
	query := `DELETE FROM public.voucher_types WHERE voucher_type_id = $1`
	_, err := db.GetPool().Exec(ctx, query, id)
	return err
}

func ListVoucherTypes(ctx context.Context, limit, offset int) ([]*models.VoucherType, error) {
	query := `
		SELECT voucher_type_id, organization_id, name, prefix, is_auto_numbered, is_reserved, parent_base_type
		FROM public.voucher_types LIMIT $1 OFFSET $2
	`
	rows, err := db.GetPool().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.VoucherType
	for rows.Next() {
		v := &models.VoucherType{}
		if err := rows.Scan(&v.VoucherTypeID, &v.OrganizationID, &v.Name, &v.Prefix, &v.IsAutoNumbered, &v.IsReserved, &v.ParentBaseType); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, nil
}

func GetVoucherTypesByOrganizationID(ctx context.Context, organizationID string) ([]*models.VoucherType, error) {
	query := `
		SELECT voucher_type_id, organization_id, name, prefix, is_auto_numbered, is_reserved, parent_base_type
		FROM public.voucher_types WHERE organization_id = $1
	`
	rows, err := db.GetPool().Query(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.VoucherType
	for rows.Next() {
		v := &models.VoucherType{}
		if err := rows.Scan(&v.VoucherTypeID, &v.OrganizationID, &v.Name, &v.Prefix, &v.IsAutoNumbered, &v.IsReserved, &v.ParentBaseType); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, nil
}

type defaultVoucherType struct {
	Name           string
	Prefix         string
	ParentBaseType string
}

var defaultVoucherTypesList = []defaultVoucherType{
	{Name: "Contra", Prefix: "CON/", ParentBaseType: "CONTRA"},
	{Name: "Payment", Prefix: "PAY/", ParentBaseType: "PAYMENT"},
	{Name: "Receipt", Prefix: "REC/", ParentBaseType: "RECEIPT"},
	{Name: "Journal", Prefix: "JRN/", ParentBaseType: "JOURNAL"},
	{Name: "Sales", Prefix: "SAL/", ParentBaseType: "SALES"},
	{Name: "Purchase", Prefix: "PUR/", ParentBaseType: "PURCHASE"},
	{Name: "Credit Note", Prefix: "CN/", ParentBaseType: "CREDIT_NOTE"},
	{Name: "Debit Note", Prefix: "DN/", ParentBaseType: "DEBIT_NOTE"},
	{Name: "Sales Order", Prefix: "SO/", ParentBaseType: "SALES_ORDER"},
	{Name: "Purchase Order", Prefix: "PO/", ParentBaseType: "PURCHASE_ORDER"},
	{Name: "Delivery Note", Prefix: "DN/", ParentBaseType: "DELIVERY_NOTE"},
	{Name: "Receipt Note", Prefix: "RN/", ParentBaseType: "RECEIPT_NOTE"},
	{Name: "Physical Stock", Prefix: "PHY/", ParentBaseType: "PHYSICAL_STOCK"},
	{Name: "Stock Journal", Prefix: "STJ/", ParentBaseType: "STOCK_JOURNAL"},
	{Name: "Rejections In", Prefix: "REJ-IN/", ParentBaseType: "REJECTIONS_IN"},
	{Name: "Rejections Out", Prefix: "REJ-OUT/", ParentBaseType: "REJECTIONS_OUT"},
	{Name: "Material In", Prefix: "MAT-IN/", ParentBaseType: "MATERIAL_IN"},
	{Name: "Material Out", Prefix: "MAT-OUT/", ParentBaseType: "MATERIAL_OUT"},
	{Name: "Memorandum", Prefix: "MEM/", ParentBaseType: "MEMORANDUM"},
	{Name: "Reversing Journal", Prefix: "REV/", ParentBaseType: "REVERSING_JOURNAL"},
}

func PopulateDefaultVoucherTypes(ctx context.Context, tx pgx.Tx, organizationID string) error {
	for _, vt := range defaultVoucherTypesList {
		query := `
			INSERT INTO public.voucher_types (organization_id, name, prefix, is_auto_numbered, is_reserved, parent_base_type)
			VALUES ($1, $2, $3, TRUE, TRUE, $4)
		`
		_, err := tx.Exec(ctx, query, organizationID, vt.Name, vt.Prefix, vt.ParentBaseType)
		if err != nil {
			return err
		}
	}
	return nil
}

func EnsureGlobalVoucherTypes(ctx context.Context) error {
	const globalOrgID = "00000000-0000-0000-0000-000000000000"

	// 1. Ensure the global organization exists
	_, err := db.GetPool().Exec(ctx, `
		INSERT INTO public.organizations (organization_id, organization_name, slug)
		VALUES ($1, 'System Global Organization', 'system-global')
		ON CONFLICT (organization_id) DO NOTHING
	`, globalOrgID)
	if err != nil {
		return fmt.Errorf("failed to ensure system global organization: %w", err)
	}

	// 2. Populate standard voucher types if they don't exist
	for _, vt := range defaultVoucherTypesList {
		var exists bool
		err = db.GetPool().QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM public.voucher_types 
				WHERE organization_id = $1 AND name = $2
			)
		`, globalOrgID, vt.Name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check existence of global voucher type %s: %w", vt.Name, err)
		}

		if !exists {
			query := `
				INSERT INTO public.voucher_types (organization_id, name, prefix, is_auto_numbered, is_reserved, parent_base_type)
				VALUES ($1, $2, $3, TRUE, TRUE, $4)
			`
			_, err = db.GetPool().Exec(ctx, query, globalOrgID, vt.Name, vt.Prefix, vt.ParentBaseType)
			if err != nil {
				return fmt.Errorf("failed to insert global voucher type %s: %w", vt.Name, err)
			}
		}
	}

	return nil
}

