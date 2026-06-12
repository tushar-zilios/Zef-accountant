package db

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"accountant/src/internal/logger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pool *pgxpool.Pool
	once sync.Once
)

// InitDB initializes the database connection pool using the provided database URL.
func InitDB(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	var err error
	logger.LogDB("Initializing database pool...")
	once.Do(func() {
		if dbURL == "" {
			err = fmt.Errorf("database URL is empty")
			logger.LogDB("DB initialization failed: database URL is empty")
			return
		}

		config, parseErr := pgxpool.ParseConfig(dbURL)
		if parseErr != nil {
			err = fmt.Errorf("failed to parse database URL: %w", parseErr)
			logger.LogDB("DB initialization failed to parse database URL: %v", parseErr)
			return
		}

		// Disable prepared statements to support connection poolers (PgBouncer/Supavisor)
		config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

		var p *pgxpool.Pool
		retryErr := retryWithExponentialBackoff(ctx, 5, 1*time.Second, 30*time.Second, func() error {
			var connErr error
			p, connErr = pgxpool.NewWithConfig(ctx, config)
			if connErr != nil {
				return fmt.Errorf("failed to connect to database: %w", connErr)
			}

			// Test connection
			if pingErr := p.Ping(ctx); pingErr != nil {
				p.Close()
				return fmt.Errorf("failed to ping database: %w", pingErr)
			}
			return nil
		}, func(format string, args ...any) {
			logger.LogDB(format, args...)
		})

		if retryErr != nil {
			err = fmt.Errorf("database initialization failed after retries: %w", retryErr)
			logger.LogDB("DB initialization failed: %v", retryErr)
			return
		}

		pool = p
		logger.LogDB("DB connection pool initialized successfully.")

		// Execute 001_init.sql migrations if they exist
		if migrationData, readErr := os.ReadFile("migrations/001_init.sql"); readErr == nil {
			logger.LogDB("Executing migrations/001_init.sql...")
			if _, execErr := p.Exec(ctx, string(migrationData)); execErr != nil {
				logger.LogDB("Warning: failed to execute migrations/001_init.sql: %v", execErr)
			} else {
				logger.LogDB("migrations/001_init.sql executed successfully.")
			}
		} else {
			logger.LogDB("Warning: migrations/001_init.sql not found: %v", readErr)
		}

		// Dynamic migration to add voucher_types columns
		_, execErr := p.Exec(ctx, `
			ALTER TABLE public.voucher_types ADD COLUMN IF NOT EXISTS is_reserved BOOLEAN DEFAULT false;
			ALTER TABLE public.voucher_types ADD COLUMN IF NOT EXISTS parent_base_type VARCHAR;
		`)
		if execErr != nil {
			logger.LogDB("Warning: failed to run table alter migrations for voucher_types: %v", execErr)
		}

		// Dynamic migration to add HSN code and GST rate to stock_items and stock_groups
		_, execErr2 := p.Exec(ctx, `
			ALTER TABLE public.stock_items ADD COLUMN IF NOT EXISTS hsn_code VARCHAR DEFAULT '';
			ALTER TABLE public.stock_items ADD COLUMN IF NOT EXISTS gst_rate NUMERIC(5,2) DEFAULT 0;
			ALTER TABLE public.stock_groups ADD COLUMN IF NOT EXISTS hsn_code VARCHAR DEFAULT '';
			ALTER TABLE public.stock_groups ADD COLUMN IF NOT EXISTS gst_rate NUMERIC(5,2) DEFAULT 0;
		`)
		if execErr2 != nil {
			logger.LogDB("Warning: failed to run table alter migrations for stock_items/stock_groups: %v", execErr2)
		}

		// Dynamic migration to create e_invoices and e_way_bills tables
		_, execErr3 := p.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS public.e_invoices (
				einvoice_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				voucher_id     UUID NOT NULL REFERENCES public.vouchers(voucher_id) ON DELETE CASCADE,
				organization_id UUID NOT NULL,
				irn            VARCHAR(64) NOT NULL,
				ack_no         VARCHAR(20) NOT NULL,
				ack_date       TIMESTAMPTZ NOT NULL,
				seller_gstin   VARCHAR(15) NOT NULL,
				buyer_gstin    VARCHAR(15) DEFAULT '',
				invoice_no     VARCHAR(50) NOT NULL,
				invoice_date   VARCHAR(20) NOT NULL,
				total_value    NUMERIC(15,2) DEFAULT 0,
				cgst           NUMERIC(15,2) DEFAULT 0,
				sgst           NUMERIC(15,2) DEFAULT 0,
				igst           NUMERIC(15,2) DEFAULT 0,
				status         VARCHAR(20) DEFAULT 'GENERATED',
				created_at     TIMESTAMPTZ DEFAULT NOW()
			);
			CREATE TABLE IF NOT EXISTS public.e_way_bills (
				eway_bill_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				voucher_id       UUID NOT NULL REFERENCES public.vouchers(voucher_id) ON DELETE CASCADE,
				organization_id  UUID NOT NULL,
				ewb_no           VARCHAR(20) NOT NULL,
				ewb_date         TIMESTAMPTZ NOT NULL,
				valid_upto       VARCHAR(25) DEFAULT '',
				seller_gstin     VARCHAR(15) NOT NULL,
				buyer_gstin      VARCHAR(15) DEFAULT '',
				transporter_id   VARCHAR(15) DEFAULT '',
				transporter_name VARCHAR(100) DEFAULT '',
				vehicle_no       VARCHAR(15) DEFAULT '',
				vehicle_type     VARCHAR(10) DEFAULT 'R',
				dispatch_from    TEXT DEFAULT '',
				ship_to          TEXT DEFAULT '',
				distance_km      INT DEFAULT 0,
				total_value      NUMERIC(15,2) DEFAULT 0,
				status           VARCHAR(20) DEFAULT 'GENERATED',
				created_at       TIMESTAMPTZ DEFAULT NOW()
			);
		`)
		if execErr3 != nil {
			logger.LogDB("Warning: failed to create e_invoices/e_way_bills tables: %v", execErr3)
		}

		// Add fiscal_year_id to e_invoices and e_way_bills
		_, execErr4 := p.Exec(ctx, `
			ALTER TABLE public.e_invoices
				ADD COLUMN IF NOT EXISTS fiscal_year_id UUID REFERENCES public.fiscal_years(fiscal_year_id) ON DELETE SET NULL;
			ALTER TABLE public.e_way_bills
				ADD COLUMN IF NOT EXISTS fiscal_year_id UUID REFERENCES public.fiscal_years(fiscal_year_id) ON DELETE SET NULL;
		`)
		if execErr4 != nil {
			logger.LogDB("Warning: failed to add fiscal_year_id to e_invoices/e_way_bills: %v", execErr4)
		}

		// Multi-godown management
		_, execErr5 := p.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS public.godowns (
				godown_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				organization_id UUID NOT NULL REFERENCES public.organizations(organization_id) ON DELETE CASCADE,
				name VARCHAR NOT NULL,
				type VARCHAR NOT NULL DEFAULT 'warehouse',
				address TEXT DEFAULT '',
				description TEXT DEFAULT '',
				created_at TIMESTAMPTZ DEFAULT NOW()
			);
			CREATE TABLE IF NOT EXISTS public.stock_transfers (
				transfer_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				organization_id UUID NOT NULL REFERENCES public.organizations(organization_id) ON DELETE CASCADE,
				from_godown_id UUID REFERENCES public.godowns(godown_id) ON DELETE SET NULL,
				to_godown_id UUID REFERENCES public.godowns(godown_id) ON DELETE SET NULL,
				stock_item_id UUID NOT NULL REFERENCES public.stock_items(stock_item_id) ON DELETE CASCADE,
				quantity NUMERIC(15,4) NOT NULL,
				rate NUMERIC(15,4) NOT NULL DEFAULT 0,
				transfer_date DATE NOT NULL DEFAULT CURRENT_DATE,
				remarks TEXT DEFAULT '',
				created_at TIMESTAMPTZ DEFAULT NOW()
			);
			ALTER TABLE public.inventory_entries ADD COLUMN IF NOT EXISTS godown_id UUID REFERENCES public.godowns(godown_id) ON DELETE SET NULL;
		`)
		if execErr5 != nil {
			logger.LogDB("Warning: failed to create godowns/stock_transfers tables: %v", execErr5)
		}

		_, execErr6 := p.Exec(ctx, `
			ALTER TABLE public.ledgers ADD COLUMN IF NOT EXISTS gst_type VARCHAR;
		`)
		if execErr6 != nil {
			logger.LogDB("Warning: failed to add gst_type to ledgers: %v", execErr6)
		}

		// Extensions catalog and workspace-level extension subscriptions
		_, execErr7 := p.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS public.extensions (
				extension_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				name                     VARCHAR NOT NULL UNIQUE,
				display_name             VARCHAR NOT NULL,
				description              TEXT DEFAULT '',
				price_monthly            BIGINT NOT NULL DEFAULT 0,
				price_yearly             BIGINT NOT NULL DEFAULT 0,
				razorpay_plan_monthly_id VARCHAR DEFAULT '',
				razorpay_plan_yearly_id  VARCHAR DEFAULT '',
				is_active                BOOLEAN NOT NULL DEFAULT true,
				created_at               TIMESTAMPTZ DEFAULT NOW()
			);
			CREATE TABLE IF NOT EXISTS public.workspace_extensions (
				workspace_extension_id  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				workspace_id            UUID NOT NULL,
				extension_id            UUID NOT NULL REFERENCES public.extensions(extension_id) ON DELETE CASCADE,
				razorpay_subscription_id VARCHAR DEFAULT '',
				razorpay_customer_id    VARCHAR DEFAULT '',
				short_url               VARCHAR DEFAULT '',
				status                  VARCHAR NOT NULL DEFAULT 'created',
				billing_frequency       VARCHAR NOT NULL DEFAULT 'monthly',
				current_period_end      TIMESTAMPTZ,
				canceled_at             TIMESTAMPTZ,
				created_at              TIMESTAMPTZ DEFAULT NOW(),
				updated_at              TIMESTAMPTZ DEFAULT NOW(),
				UNIQUE(workspace_id, extension_id)
			);
		`)
		if execErr7 != nil {
			logger.LogDB("Warning: failed to create extensions/workspace_extensions tables: %v", execErr7)
		}

		// Drop all CTO tables from the main Zef DB — they now live in CTO_DATABASE_URL
		_, execErrDropCTO := pool.Exec(ctx, `
			DROP TABLE IF EXISTS public.cto_schema_snapshots   CASCADE;
			DROP TABLE IF EXISTS public.cto_saved_queries      CASCADE;
			DROP TABLE IF EXISTS public.cto_sql_history        CASCADE;
			DROP TABLE IF EXISTS public.cto_connection_health  CASCADE;
			DROP TABLE IF EXISTS public.cto_database_credentials CASCADE;
			DROP TABLE IF EXISTS public.cto_ideate_messages    CASCADE;
			DROP TABLE IF EXISTS public.cto_database_projects  CASCADE;
		`)
		if execErrDropCTO != nil {
			logger.LogDB("Warning: failed to drop CTO tables from main DB: %v", execErrDropCTO)
		}

		// Create companies and organization_to_company tables if missing
		_, execErrCompanies := pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS public.companies (
				company_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				name                 VARCHAR(150),
				legal_name           VARCHAR,
				tax_identifier       VARCHAR,
				base_currency        VARCHAR(3) DEFAULT 'INR',
				alias                VARCHAR,
				account_number       VARCHAR,
				ifsc_code            VARCHAR,
				bank_name            VARCHAR,
				branch               VARCHAR,
				bsr_code             VARCHAR,
				pan_number           VARCHAR,
				type_of_registration VARCHAR,
				mailing_name         VARCHAR,
				address              TEXT,
				state                VARCHAR,
				country              VARCHAR,
				pin_code             VARCHAR,
				created_at           TIMESTAMPTZ DEFAULT NOW(),
				updated_at           TIMESTAMPTZ DEFAULT NOW()
			);
			CREATE TABLE IF NOT EXISTS public.organization_to_company (
				mapping_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				organization_id UUID NOT NULL REFERENCES public.organizations(organization_id) ON DELETE CASCADE,
				company_id      UUID NOT NULL REFERENCES public.companies(company_id) ON DELETE CASCADE,
				UNIQUE (organization_id, company_id)
			);
		`)
		if execErrCompanies != nil {
			logger.LogDB("Warning: failed to create companies/organization_to_company tables: %v", execErrCompanies)
		}

		// RBAC: scope role assignments to workspace/org
		_, execErr10 := pool.Exec(ctx, `
			ALTER TABLE public.roles_to_users
				ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES public.workspace(workspace_id) ON DELETE CASCADE,
				ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES public.organizations(organization_id) ON DELETE CASCADE;
			CREATE UNIQUE INDEX IF NOT EXISTS roles_to_users_unique_scope
				ON public.roles_to_users (user_id, role_id, COALESCE(workspace_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(organization_id, '00000000-0000-0000-0000-000000000000'::uuid));
		`)
		if execErr10 != nil {
			logger.LogDB("Warning: failed to add scope columns to roles_to_users: %v", execErr10)
		}

	})

	return pool, err
}

// GetPool returns the initialized database pool. It panics if InitDB has not been successfully called.
func GetPool() *pgxpool.Pool {
	if pool == nil {
		panic("database pool is not initialized. Call InitDB first")
	}
	return pool
}

// SeedRBACData inserts system resources, actions, permissions, and roles.
// It is idempotent (ON CONFLICT DO NOTHING) and must be called after InitDB.
// Called on every startup so data is restored after a truncate without needing a restart.
func SeedRBACData(ctx context.Context) {
	_, err := GetPool().Exec(ctx, `
		INSERT INTO public.resources (resource_id, code, title) VALUES
			('00000001-0000-0000-0000-000000000001', 'task',      'Task'),
			('00000001-0000-0000-0000-000000000002', 'reminder',  'Reminder'),
			('00000001-0000-0000-0000-000000000003', 'knowledge', 'Knowledge'),
			('00000001-0000-0000-0000-000000000004', 'cfo',       'CFO / Accounting'),
			('00000001-0000-0000-0000-000000000005', 'workspace', 'Workspace'),
			('00000001-0000-0000-0000-000000000006', 'user',      'User')
		ON CONFLICT DO NOTHING;

		INSERT INTO public.actions (action_id, code, title) VALUES
			('00000002-0000-0000-0000-000000000001', 'create', 'Create'),
			('00000002-0000-0000-0000-000000000002', 'read',   'Read'),
			('00000002-0000-0000-0000-000000000003', 'update', 'Update'),
			('00000002-0000-0000-0000-000000000004', 'delete', 'Delete'),
			('00000002-0000-0000-0000-000000000005', 'manage', 'Manage')
		ON CONFLICT DO NOTHING;

		INSERT INTO public.permissions (permission_id, resource_id, action_id, code)
		SELECT
			('00000003-' || lpad((ROW_NUMBER() OVER ())::text, 4, '0') || '-0000-0000-000000000000')::uuid,
			r.resource_id, a.action_id,
			r.code || '.' || a.code
		FROM public.resources r CROSS JOIN public.actions a
		WHERE r.resource_id::text LIKE '00000001-%'
		  AND a.action_id::text  LIKE '00000002-%'
		ON CONFLICT DO NOTHING;

		INSERT INTO public.roles (role_id, is_system, code, name) VALUES
			('00000004-0000-0000-0000-000000000001', true, 'owner',             'Owner'),
			('00000004-0000-0000-0000-000000000002', true, 'admin',             'Admin'),
			('00000004-0000-0000-0000-000000000003', true, 'member',            'Member'),
			('00000004-0000-0000-0000-000000000004', true, 'viewer',            'Viewer'),
			('00000004-0000-0000-0000-000000000005', true, 'accountant',        'Accountant'),
			('00000004-0000-0000-0000-000000000006', true, 'organization_admin','Organization Admin')
		ON CONFLICT DO NOTHING;

		INSERT INTO public.roles_to_permissions (role_id, permissions_id)
		SELECT '00000004-0000-0000-0000-000000000001', permission_id
		FROM public.permissions WHERE code LIKE '%.%'
		ON CONFLICT DO NOTHING;

		INSERT INTO public.roles_to_permissions (role_id, permissions_id)
		SELECT '00000004-0000-0000-0000-000000000002', permission_id
		FROM public.permissions WHERE code LIKE '%.%'
		  AND code NOT IN ('user.delete', 'workspace.delete')
		ON CONFLICT DO NOTHING;

		INSERT INTO public.roles_to_permissions (role_id, permissions_id)
		SELECT '00000004-0000-0000-0000-000000000003', permission_id
		FROM public.permissions WHERE code IN (
			'task.create','task.read','task.update','task.delete',
			'reminder.create','reminder.read','reminder.update','reminder.delete',
			'knowledge.create','knowledge.read','knowledge.update'
		)
		ON CONFLICT DO NOTHING;

		INSERT INTO public.roles_to_permissions (role_id, permissions_id)
		SELECT '00000004-0000-0000-0000-000000000004', permission_id
		FROM public.permissions WHERE code LIKE '%.read'
		ON CONFLICT DO NOTHING;

		INSERT INTO public.roles_to_permissions (role_id, permissions_id)
		SELECT '00000004-0000-0000-0000-000000000005', permission_id
		FROM public.permissions WHERE code LIKE 'cfo.%'
		ON CONFLICT DO NOTHING;

		INSERT INTO public.roles_to_permissions (role_id, permissions_id)
		SELECT '00000004-0000-0000-0000-000000000006', permission_id
		FROM public.permissions WHERE code LIKE 'workspace.%' OR code LIKE 'user.%'
		ON CONFLICT DO NOTHING;
	`)
	if err != nil {
		logger.LogDB("Warning: failed to seed RBAC system data: %v", err)
	}
}

// CloseDB closes the connection pool.
func CloseDB() {
	if pool != nil {
		logger.LogDB("Closing database connection pool.")
		pool.Close()
	}
}

