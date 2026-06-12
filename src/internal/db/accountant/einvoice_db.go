package accountant

import (
	"context"

	"accountant/src/internal/db"
	models "accountant/src/internal/models/accountant"
)

func CreateEInvoice(ctx context.Context, e *models.EInvoice) error {
	row := db.GetPool().QueryRow(ctx, `
		INSERT INTO public.e_invoices
			(voucher_id, organization_id, fiscal_year_id, irn, ack_no, ack_date, seller_gstin, buyer_gstin,
			 invoice_no, invoice_date, total_value, cgst, sgst, igst, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING einvoice_id, created_at`,
		e.VoucherID, e.OrganizationID, nullableUUID(e.FiscalYearID), e.IRN, e.AckNo, e.AckDate,
		e.SellerGSTIN, e.BuyerGSTIN, e.InvoiceNo, e.InvoiceDate,
		e.TotalValue, e.CGST, e.SGST, e.IGST, e.Status,
	)
	return row.Scan(&e.EInvoiceID, &e.CreatedAt)
}

func GetEInvoiceByVoucherID(ctx context.Context, voucherID string) (*models.EInvoice, error) {
	e := &models.EInvoice{}
	var fyID *string
	err := db.GetPool().QueryRow(ctx, `
		SELECT einvoice_id, voucher_id, organization_id, fiscal_year_id, irn, ack_no, ack_date,
		       seller_gstin, buyer_gstin, invoice_no, invoice_date,
		       total_value, cgst, sgst, igst, status, created_at
		FROM public.e_invoices WHERE voucher_id = $1`, voucherID,
	).Scan(&e.EInvoiceID, &e.VoucherID, &e.OrganizationID, &fyID, &e.IRN, &e.AckNo, &e.AckDate,
		&e.SellerGSTIN, &e.BuyerGSTIN, &e.InvoiceNo, &e.InvoiceDate,
		&e.TotalValue, &e.CGST, &e.SGST, &e.IGST, &e.Status, &e.CreatedAt)
	if fyID != nil {
		e.FiscalYearID = *fyID
	}
	return e, err
}

func ListEInvoicesByFiscalYear(ctx context.Context, fiscalYearID, organizationID string) ([]*models.EInvoice, error) {
	rows, err := db.GetPool().Query(ctx, `
		SELECT einvoice_id, voucher_id, organization_id, fiscal_year_id, irn, ack_no, ack_date,
		       seller_gstin, buyer_gstin, invoice_no, invoice_date,
		       total_value, cgst, sgst, igst, status, created_at
		FROM public.e_invoices
		WHERE fiscal_year_id = $1 AND organization_id = $2
		ORDER BY created_at DESC`,
		fiscalYearID, organizationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.EInvoice
	for rows.Next() {
		e := &models.EInvoice{}
		var fyID *string
		if err := rows.Scan(&e.EInvoiceID, &e.VoucherID, &e.OrganizationID, &fyID, &e.IRN, &e.AckNo, &e.AckDate,
			&e.SellerGSTIN, &e.BuyerGSTIN, &e.InvoiceNo, &e.InvoiceDate,
			&e.TotalValue, &e.CGST, &e.SGST, &e.IGST, &e.Status, &e.CreatedAt); err != nil {
			return nil, err
		}
		if fyID != nil {
			e.FiscalYearID = *fyID
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

func CreateEWayBill(ctx context.Context, b *models.EWayBill) error {
	row := db.GetPool().QueryRow(ctx, `
		INSERT INTO public.e_way_bills
			(voucher_id, organization_id, fiscal_year_id, ewb_no, ewb_date, valid_upto,
			 seller_gstin, buyer_gstin, transporter_id, transporter_name,
			 vehicle_no, vehicle_type, dispatch_from, ship_to, distance_km, total_value, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		RETURNING eway_bill_id, created_at`,
		b.VoucherID, b.OrganizationID, nullableUUID(b.FiscalYearID), b.EWBNo, b.EWBDate, b.ValidUpto,
		b.SellerGSTIN, b.BuyerGSTIN, b.TransporterID, b.TransporterName,
		b.VehicleNo, b.VehicleType, b.DispatchFrom, b.ShipTo, b.Distance,
		b.TotalValue, b.Status,
	)
	return row.Scan(&b.EWayBillID, &b.CreatedAt)
}

func GetEWayBillByVoucherID(ctx context.Context, voucherID string) (*models.EWayBill, error) {
	b := &models.EWayBill{}
	var fyID *string
	err := db.GetPool().QueryRow(ctx, `
		SELECT eway_bill_id, voucher_id, organization_id, fiscal_year_id, ewb_no, ewb_date, valid_upto,
		       seller_gstin, buyer_gstin, transporter_id, transporter_name,
		       vehicle_no, vehicle_type, dispatch_from, ship_to, distance_km, total_value, status, created_at
		FROM public.e_way_bills WHERE voucher_id = $1`, voucherID,
	).Scan(&b.EWayBillID, &b.VoucherID, &b.OrganizationID, &fyID, &b.EWBNo, &b.EWBDate, &b.ValidUpto,
		&b.SellerGSTIN, &b.BuyerGSTIN, &b.TransporterID, &b.TransporterName,
		&b.VehicleNo, &b.VehicleType, &b.DispatchFrom, &b.ShipTo, &b.Distance,
		&b.TotalValue, &b.Status, &b.CreatedAt)
	if fyID != nil {
		b.FiscalYearID = *fyID
	}
	return b, err
}

func ListEWayBillsByFiscalYear(ctx context.Context, fiscalYearID, organizationID string) ([]*models.EWayBill, error) {
	rows, err := db.GetPool().Query(ctx, `
		SELECT eway_bill_id, voucher_id, organization_id, fiscal_year_id, ewb_no, ewb_date, valid_upto,
		       seller_gstin, buyer_gstin, transporter_id, transporter_name,
		       vehicle_no, vehicle_type, dispatch_from, ship_to, distance_km, total_value, status, created_at
		FROM public.e_way_bills
		WHERE fiscal_year_id = $1 AND organization_id = $2
		ORDER BY created_at DESC`,
		fiscalYearID, organizationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.EWayBill
	for rows.Next() {
		b := &models.EWayBill{}
		var fyID *string
		if err := rows.Scan(&b.EWayBillID, &b.VoucherID, &b.OrganizationID, &fyID, &b.EWBNo, &b.EWBDate, &b.ValidUpto,
			&b.SellerGSTIN, &b.BuyerGSTIN, &b.TransporterID, &b.TransporterName,
			&b.VehicleNo, &b.VehicleType, &b.DispatchFrom, &b.ShipTo, &b.Distance,
			&b.TotalValue, &b.Status, &b.CreatedAt); err != nil {
			return nil, err
		}
		if fyID != nil {
			b.FiscalYearID = *fyID
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

// nullableUUID returns nil for an empty string so the DB stores NULL instead of an invalid UUID.
func nullableUUID(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}
