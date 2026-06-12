-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- ENUMS
-- ============================================================
DO $$ BEGIN
    CREATE TYPE public.account_class AS ENUM (
        'ASSET', 'LIABILITY', 'EQUITY', 'INCOME', 'EXPENSE'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ============================================================
-- organizations (stub — FK dependency for accountant tables)
-- ============================================================
CREATE TABLE IF NOT EXISTS public.organizations (
    organization_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_name TEXT NOT NULL,
    slug              TEXT,
    logo_url          TEXT,
    created_at        TIMESTAMPTZ DEFAULT now(),
    updated_at        TIMESTAMPTZ DEFAULT now()
);

-- ============================================================
-- workspace (stub — FK dependency for roles_to_users)
-- ============================================================
CREATE TABLE IF NOT EXISTS public.workspace (
    workspace_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT,
    created_at   TIMESTAMPTZ DEFAULT now()
);


-- ============================================================
-- users (stub — FK dependency for vouchers.posted_by)
-- ============================================================
CREATE TABLE IF NOT EXISTS public.users (
    user_id       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    first_name    VARCHAR,
    last_name     VARCHAR,
    email         VARCHAR UNIQUE,
    password_hash VARCHAR,
    created_at    TIMESTAMPTZ DEFAULT now(),
    updated_at    TIMESTAMPTZ DEFAULT now()
);

-- ============================================================
-- RBAC stubs (needed by JWT middleware permission checks)
-- ============================================================
CREATE TABLE IF NOT EXISTS public.roles (
    role_id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    is_system       BOOLEAN DEFAULT false,
    organization_id UUID,
    code            VARCHAR,
    name            VARCHAR NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.permissions (
    permission_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id   UUID,
    action_id     UUID,
    code          VARCHAR
);

CREATE TABLE IF NOT EXISTS public.roles_to_permissions (
    mapping_id     UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_id        UUID REFERENCES public.roles(role_id) ON DELETE CASCADE,
    permissions_id UUID REFERENCES public.permissions(permission_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS public.roles_to_users (
    mapping_id      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID REFERENCES public.users(user_id) ON DELETE CASCADE,
    role_id         UUID REFERENCES public.roles(role_id) ON DELETE CASCADE,
    workspace_id    UUID,
    organization_id UUID,
    created_at      TIMESTAMPTZ DEFAULT now()
);

-- ============================================================
-- ACCOUNTANT TABLES
-- ============================================================

-- fiscal_years
CREATE TABLE IF NOT EXISTS public.fiscal_years (
    fiscal_year_id  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES public.organizations(organization_id) ON DELETE CASCADE,
    year_label      VARCHAR(50) NOT NULL,
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    is_closed       BOOLEAN DEFAULT false
);

-- account_groups
CREATE TABLE IF NOT EXISTS public.account_groups (
    account_group_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id  UUID NOT NULL REFERENCES public.organizations(organization_id) ON DELETE CASCADE,
    parent_id        UUID REFERENCES public.account_groups(account_group_id) ON DELETE SET NULL,
    name             VARCHAR(150) NOT NULL,
    classification   public.account_class NOT NULL,
    is_reserved      BOOLEAN DEFAULT false
);

-- voucher_types
CREATE TABLE IF NOT EXISTS public.voucher_types (
    voucher_type_id  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id  UUID NOT NULL REFERENCES public.organizations(organization_id) ON DELETE CASCADE,
    name             VARCHAR(50) NOT NULL,
    prefix           VARCHAR(10),
    is_auto_numbered BOOLEAN DEFAULT true,
    is_reserved      BOOLEAN DEFAULT false,
    parent_base_type VARCHAR
);

-- ledgers
CREATE TABLE IF NOT EXISTS public.ledgers (
    ledger_id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id           UUID,
    group_id             UUID NOT NULL REFERENCES public.account_groups(account_group_id) ON DELETE RESTRICT,
    name                 VARCHAR(150) NOT NULL,
    currency             VARCHAR(3) NOT NULL DEFAULT 'INR',
    opening_balance      NUMERIC(18,4) NOT NULL DEFAULT 0.0000,
    opening_balance_type VARCHAR(2) NOT NULL,
    is_active            BOOLEAN DEFAULT true,
    organization_id      UUID REFERENCES public.organizations(organization_id) ON DELETE CASCADE,
    legal_name           VARCHAR,
    tax_identifier       VARCHAR,
    base_currency        VARCHAR,
    alias                VARCHAR,
    account_no           VARCHAR,
    ifs_code             VARCHAR,
    bank_name            VARCHAR,
    branch               VARCHAR,
    bsr_code             VARCHAR,
    pan_it_no            VARCHAR,
    type_of_reg          VARCHAR,
    mailing_name         VARCHAR,
    address              TEXT,
    state                VARCHAR,
    country              VARCHAR,
    pin_code             VARCHAR,
    gst_type             VARCHAR
);

-- vouchers
CREATE TABLE IF NOT EXISTS public.vouchers (
    voucher_id      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id      UUID,
    fiscal_year_id  UUID NOT NULL REFERENCES public.fiscal_years(fiscal_year_id) ON DELETE RESTRICT,
    voucher_type_id UUID NOT NULL REFERENCES public.voucher_types(voucher_type_id) ON DELETE RESTRICT,
    voucher_number  VARCHAR(100) NOT NULL,
    date            DATE NOT NULL,
    narration       TEXT,
    posted_by       UUID REFERENCES public.users(user_id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    organization_id UUID REFERENCES public.organizations(organization_id) ON DELETE CASCADE
);

-- journal_entries
CREATE TABLE IF NOT EXISTS public.journal_entries (
    journal_entry_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    voucher_id       UUID NOT NULL REFERENCES public.vouchers(voucher_id) ON DELETE CASCADE,
    ledger_id        UUID NOT NULL REFERENCES public.ledgers(ledger_id) ON DELETE RESTRICT,
    debit            NUMERIC(18,4) NOT NULL DEFAULT 0.0000,
    credit           NUMERIC(18,4) NOT NULL DEFAULT 0.0000,
    currency_rate    NUMERIC(12,6) NOT NULL DEFAULT 1.000000,
    narration        TEXT,
    sequence_order   INTEGER NOT NULL
);

-- stock_groups
CREATE TABLE IF NOT EXISTS public.stock_groups (
    stock_group_id  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES public.organizations(organization_id) ON DELETE CASCADE,
    name            VARCHAR(150) NOT NULL,
    parent_id       UUID REFERENCES public.stock_groups(stock_group_id) ON DELETE SET NULL,
    hsn_code        VARCHAR DEFAULT '',
    gst_rate        NUMERIC(5,2) DEFAULT 0
);

-- stock_items
CREATE TABLE IF NOT EXISTS public.stock_items (
    stock_item_id     UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id          UUID REFERENCES public.stock_groups(stock_group_id) ON DELETE SET NULL,
    name              VARCHAR(150) NOT NULL,
    unit_of_measure   VARCHAR(20) NOT NULL,
    opening_qty       NUMERIC(18,4) NOT NULL DEFAULT 0.0000,
    opening_valuation NUMERIC(18,4) NOT NULL DEFAULT 0.0000,
    costing_method    VARCHAR(20) NOT NULL DEFAULT 'FIFO',
    hsn_code          VARCHAR DEFAULT '',
    gst_rate          NUMERIC(5,2) DEFAULT 0
);

-- godowns
CREATE TABLE IF NOT EXISTS public.godowns (
    godown_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES public.organizations(organization_id) ON DELETE CASCADE,
    name            VARCHAR NOT NULL,
    type            VARCHAR NOT NULL DEFAULT 'warehouse',
    address         TEXT DEFAULT '',
    description     TEXT DEFAULT '',
    created_at      TIMESTAMPTZ DEFAULT now()
);

-- inventory_entries
CREATE TABLE IF NOT EXISTS public.inventory_entries (
    inventory_entry_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    journal_entry_id   UUID NOT NULL REFERENCES public.journal_entries(journal_entry_id) ON DELETE CASCADE,
    stock_item_id      UUID NOT NULL REFERENCES public.stock_items(stock_item_id) ON DELETE RESTRICT,
    quantity           NUMERIC(18,4) NOT NULL,
    rate               NUMERIC(18,4) NOT NULL,
    amount             NUMERIC(18,4) NOT NULL,
    movement_type      VARCHAR(3) NOT NULL,
    godown_id          UUID REFERENCES public.godowns(godown_id) ON DELETE SET NULL
);

-- stock_transfers
CREATE TABLE IF NOT EXISTS public.stock_transfers (
    transfer_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES public.organizations(organization_id) ON DELETE CASCADE,
    from_godown_id  UUID REFERENCES public.godowns(godown_id) ON DELETE SET NULL,
    to_godown_id    UUID REFERENCES public.godowns(godown_id) ON DELETE SET NULL,
    stock_item_id   UUID NOT NULL REFERENCES public.stock_items(stock_item_id) ON DELETE RESTRICT,
    quantity        NUMERIC(15,4) NOT NULL,
    rate            NUMERIC(15,4) NOT NULL DEFAULT 0,
    transfer_date   DATE NOT NULL DEFAULT CURRENT_DATE,
    remarks         TEXT DEFAULT '',
    created_at      TIMESTAMPTZ DEFAULT now()
);

-- e_invoices
CREATE TABLE IF NOT EXISTS public.e_invoices (
    einvoice_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voucher_id       UUID NOT NULL REFERENCES public.vouchers(voucher_id) ON DELETE CASCADE,
    organization_id  UUID NOT NULL REFERENCES public.organizations(organization_id) ON DELETE CASCADE,
    irn              VARCHAR(64) NOT NULL,
    ack_no           VARCHAR(20) NOT NULL,
    ack_date         TIMESTAMPTZ NOT NULL,
    seller_gstin     VARCHAR(15) NOT NULL,
    buyer_gstin      VARCHAR(15) DEFAULT '',
    invoice_no       VARCHAR(50) NOT NULL,
    invoice_date     VARCHAR(20) NOT NULL,
    total_value      NUMERIC(15,2) DEFAULT 0,
    cgst             NUMERIC(15,2) DEFAULT 0,
    sgst             NUMERIC(15,2) DEFAULT 0,
    igst             NUMERIC(15,2) DEFAULT 0,
    status           VARCHAR(20) DEFAULT 'GENERATED',
    created_at       TIMESTAMPTZ DEFAULT now(),
    fiscal_year_id   UUID REFERENCES public.fiscal_years(fiscal_year_id) ON DELETE SET NULL
);

-- e_way_bills
CREATE TABLE IF NOT EXISTS public.e_way_bills (
    eway_bill_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voucher_id        UUID NOT NULL REFERENCES public.vouchers(voucher_id) ON DELETE CASCADE,
    organization_id   UUID NOT NULL REFERENCES public.organizations(organization_id) ON DELETE CASCADE,
    ewb_no            VARCHAR(20) NOT NULL,
    ewb_date          TIMESTAMPTZ NOT NULL,
    valid_upto        VARCHAR(25) DEFAULT '',
    seller_gstin      VARCHAR(15) NOT NULL,
    buyer_gstin       VARCHAR(15) DEFAULT '',
    transporter_id    VARCHAR(15) DEFAULT '',
    transporter_name  VARCHAR(100) DEFAULT '',
    vehicle_no        VARCHAR(15) DEFAULT '',
    vehicle_type      VARCHAR(10) DEFAULT 'R',
    dispatch_from     TEXT DEFAULT '',
    ship_to           TEXT DEFAULT '',
    distance_km       INTEGER DEFAULT 0,
    total_value       NUMERIC(15,2) DEFAULT 0,
    status            VARCHAR(20) DEFAULT 'GENERATED',
    created_at        TIMESTAMPTZ DEFAULT now(),
    fiscal_year_id    UUID REFERENCES public.fiscal_years(fiscal_year_id) ON DELETE SET NULL
);
