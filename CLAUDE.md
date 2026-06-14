# Zef-accountant

## DATABASE OWNERSHIP
This service owns **one specific Supabase PostgreSQL database**:
- Project ID: `wkahcihmyffxsejniftm` (ap-south-1)
- Env var: `DATABASE_URL` in `Zef-accountant/.env`

**DO NOT** read or modify DB schemas/queries from `Zef-backend/` or `Zef-cto/`. Those are separate services with separate databases.

## This service handles
- Accountant / CFO service (separate from backend's CFO module)
- Port 8082
- Has migrations in `migrations/` directory

Run: `cd Zef-accountant && go run accountant/main.go` (or via root `make run-accountant`)
