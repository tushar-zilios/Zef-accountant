package accountant

import (
	"context"
	"errors"
	"accountant/src/internal/db"
	"accountant/src/internal/models/accountant"
)

// --- AccountGroup Actions ---

func CreateAccountGroup(ctx context.Context, g *models.AccountGroup) error {
	query := `
		INSERT INTO public.account_groups (organization_id, parent_id, name, classification, is_reserved)
		VALUES ($1, $2, $3, $4, COALESCE($5, false))
		RETURNING account_group_id, is_reserved
	`
	return db.GetPool().QueryRow(ctx, query, g.OrganizationID, g.ParentID, g.Name, g.Classification, g.IsReserved).
		Scan(&g.AccountGroupID, &g.IsReserved)
}

func GetAccountGroupByID(ctx context.Context, id string) (*models.AccountGroup, error) {
	query := `
		SELECT account_group_id, organization_id, parent_id, name, classification, is_reserved
		FROM public.account_groups WHERE account_group_id = $1
	`
	g := &models.AccountGroup{}
	err := db.GetPool().QueryRow(ctx, query, id).
		Scan(&g.AccountGroupID, &g.OrganizationID, &g.ParentID, &g.Name, &g.Classification, &g.IsReserved)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func UpdateAccountGroup(ctx context.Context, g *models.AccountGroup) error {
	query := `
		UPDATE public.account_groups
		SET organization_id = $1, parent_id = $2, name = $3, classification = $4, is_reserved = $5
		WHERE account_group_id = $6
	`
	_, err := db.GetPool().Exec(ctx, query, g.OrganizationID, g.ParentID, g.Name, g.Classification, g.IsReserved, g.AccountGroupID)
	return err
}

func DeleteAccountGroup(ctx context.Context, id string) error {
	// 2. Check for child groups referencing this parent_id
	var childCount int
	err := db.GetPool().QueryRow(ctx, "SELECT COUNT(*) FROM public.account_groups WHERE parent_id = $1", id).Scan(&childCount)
	if err != nil {
		return err
	}
	if childCount > 0 {
		return errors.New("Cannot delete a group containing active sub-groups or ledgers. Reassign those elements first.")
	}

	// 3. Check for ledgers referencing this group_id
	var ledgerCount int
	err = db.GetPool().QueryRow(ctx, "SELECT COUNT(*) FROM public.ledgers WHERE group_id = $1", id).Scan(&ledgerCount)
	if err != nil {
		return err
	}
	if ledgerCount > 0 {
		return errors.New("Cannot delete a group containing active sub-groups or ledgers. Reassign those elements first.")
	}

	query := `DELETE FROM public.account_groups WHERE account_group_id = $1`
	_, err = db.GetPool().Exec(ctx, query, id)
	return err
}

func ListAccountGroups(ctx context.Context, limit, offset int) ([]*models.AccountGroup, error) {
	query := `
		SELECT account_group_id, organization_id, parent_id, name, classification, is_reserved
		FROM public.account_groups LIMIT $1 OFFSET $2
	`
	rows, err := db.GetPool().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.AccountGroup
	for rows.Next() {
		g := &models.AccountGroup{}
		if err := rows.Scan(&g.AccountGroupID, &g.OrganizationID, &g.ParentID, &g.Name, &g.Classification, &g.IsReserved); err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, nil
}

func GetAccountGroupsByOrganizationID(ctx context.Context, organizationID string) ([]*models.AccountGroup, error) {
	query := `
		SELECT account_group_id, organization_id, parent_id, name, classification, is_reserved
		FROM public.account_groups WHERE organization_id = $1
	`
	rows, err := db.GetPool().Query(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.AccountGroup
	for rows.Next() {
		g := &models.AccountGroup{}
		if err := rows.Scan(&g.AccountGroupID, &g.OrganizationID, &g.ParentID, &g.Name, &g.Classification, &g.IsReserved); err != nil {
			return nil, err
		}
		list = append(list, g)
	}

	if len(list) == 0 {
		// Start a transaction to populate standard groups
		tx, err := db.GetPool().Begin(ctx)
		if err == nil {
			defer tx.Rollback(ctx)
			if errPop := PopulateStandardAccountGroups(ctx, tx, organizationID); errPop == nil {
				if errCommit := tx.Commit(ctx); errCommit == nil {
					// Query again after successful commit
					rows2, errQ := db.GetPool().Query(ctx, query, organizationID)
					if errQ == nil {
						defer rows2.Close()
						list = []*models.AccountGroup{}
						for rows2.Next() {
							g := &models.AccountGroup{}
							if errScan := rows2.Scan(&g.AccountGroupID, &g.OrganizationID, &g.ParentID, &g.Name, &g.Classification, &g.IsReserved); errScan == nil {
								list = append(list, g)
							}
						}
					}
				}
			}
		}
	}

	return list, nil
}

func GetAccountGroupBalance(ctx context.Context, groupID string) (*models.GroupBalance, error) {
	// First fetch the classification of the group so we can calculate the normal balance
	var classification string
	err := db.GetPool().QueryRow(ctx, `SELECT classification FROM public.account_groups WHERE account_group_id = $1`, groupID).Scan(&classification)
	if err != nil {
		return nil, err
	}

	query := `
		WITH RECURSIVE group_hierarchy AS (
			-- Anchor member: select the target group
			SELECT account_group_id, parent_id, name
			FROM public.account_groups
			WHERE account_group_id = $1
			
			UNION ALL
			
			-- Recursive member: select child groups
			SELECT g.account_group_id, g.parent_id, g.name
			FROM public.account_groups g
			INNER JOIN group_hierarchy gh ON g.parent_id = gh.account_group_id
		),
		child_ledgers AS (
			-- Get all ledgers belonging to any group in the hierarchy
			SELECT 
				l.ledger_id,
				l.opening_balance,
				l.opening_balance_type
			FROM public.ledgers l
			WHERE l.group_id IN (SELECT account_group_id FROM group_hierarchy)
		),
		ledger_entries AS (
			-- Sum up debits and credits for each child ledger
			SELECT 
				l.ledger_id,
				l.opening_balance,
				l.opening_balance_type,
				COALESCE(SUM(je.debit), 0) as total_je_debit,
				COALESCE(SUM(je.credit), 0) as total_je_credit
			FROM child_ledgers l
			LEFT JOIN public.journal_entries je ON l.ledger_id = je.ledger_id
			GROUP BY l.ledger_id, l.opening_balance, l.opening_balance_type
		),
		ledger_balances AS (
			-- Compute final total_debit and total_credit for each ledger
			SELECT 
				ledger_id,
				CASE WHEN UPPER(opening_balance_type) = 'DR' THEN opening_balance ELSE 0 END + total_je_debit AS total_debit,
				CASE WHEN UPPER(opening_balance_type) = 'CR' THEN opening_balance ELSE 0 END + total_je_credit AS total_credit
			FROM ledger_entries
		)
		SELECT 
			COALESCE(SUM(total_debit), 0) AS total_debit,
			COALESCE(SUM(total_credit), 0) AS total_credit,
			COALESCE(SUM(total_debit - total_credit), 0) AS net_balance_dr,
			COALESCE(SUM(total_credit - total_debit), 0) AS net_balance_cr
		FROM ledger_balances;
	`

	balance := &models.GroupBalance{AccountGroupID: groupID}
	err = db.GetPool().QueryRow(ctx, query, groupID).
		Scan(&balance.TotalDebit, &balance.TotalCredit, &balance.NetBalanceDR, &balance.NetBalanceCR)
	if err != nil {
		return nil, err
	}

	// Calculate classification-aware balance
	switch classification {
	case "ASSET", "EXPENSE":
		balance.Balance = balance.NetBalanceDR
	case "LIABILITY", "EQUITY", "REVENUE", "INCOME":
		balance.Balance = balance.NetBalanceCR
	default:
		// Default to net Debit minus Credit
		balance.Balance = balance.NetBalanceDR
	}

	return balance, nil
}

type GroupBalanceReportItem struct {
	AccountGroupID string  `json:"account_group_id"`
	ParentID       *string `json:"parent_id"`
	Name           string  `json:"name"`
	Classification string  `json:"classification"`
	TotalDebit     float64 `json:"total_debit"`
	TotalCredit    float64 `json:"total_credit"`
	NetBalanceDR   float64 `json:"net_balance_dr"`
	NetBalanceCR   float64 `json:"net_balance_cr"`
	Balance        float64 `json:"balance"`
}

func GetCompanyAccountGroupBalances(ctx context.Context, organizationID string) ([]*GroupBalanceReportItem, error) {
	query := `
		WITH RECURSIVE
		subtree AS (
			SELECT account_group_id AS ancestor_group_id,
			       account_group_id AS descendant_group_id
			FROM public.account_groups
			WHERE organization_id = $1

			UNION ALL

			SELECT s.ancestor_group_id, g.account_group_id AS descendant_group_id
			FROM public.account_groups g
			INNER JOIN subtree s ON g.parent_id = s.descendant_group_id
			WHERE g.organization_id = $1
		),
		ledger_balances AS (
			SELECT
				l.ledger_id,
				l.group_id,
				CASE WHEN UPPER(l.opening_balance_type) = 'DR' THEN l.opening_balance ELSE 0 END
					+ COALESCE(SUM(je.debit), 0) AS total_debit,
				CASE WHEN UPPER(l.opening_balance_type) = 'CR' THEN l.opening_balance ELSE 0 END
					+ COALESCE(SUM(je.credit), 0) AS total_credit
			FROM public.ledgers l
			LEFT JOIN public.journal_entries je ON l.ledger_id = je.ledger_id
			WHERE l.organization_id = $1
			GROUP BY l.ledger_id, l.group_id, l.opening_balance, l.opening_balance_type
		),
		group_balances AS (
			SELECT
				s.ancestor_group_id AS account_group_id,
				SUM(lb.total_debit)                     AS total_debit,
				SUM(lb.total_credit)                    AS total_credit,
				SUM(lb.total_debit - lb.total_credit)   AS net_balance_dr,
				SUM(lb.total_credit - lb.total_debit)   AS net_balance_cr
			FROM subtree s
			INNER JOIN ledger_balances lb ON lb.group_id = s.descendant_group_id
			GROUP BY s.ancestor_group_id
		)
		SELECT
			g.account_group_id,
			g.parent_id,
			g.name,
			g.classification,
			COALESCE(gb.total_debit, 0)    AS total_debit,
			COALESCE(gb.total_credit, 0)   AS total_credit,
			COALESCE(gb.net_balance_dr, 0) AS net_balance_dr,
			COALESCE(gb.net_balance_cr, 0) AS net_balance_cr
		FROM public.account_groups g
		LEFT JOIN group_balances gb ON g.account_group_id = gb.account_group_id
		WHERE g.organization_id = $1;
	`

	rows, err := db.GetPool().Query(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*GroupBalanceReportItem
	for rows.Next() {
		item := &GroupBalanceReportItem{}
		err := rows.Scan(
			&item.AccountGroupID,
			&item.ParentID,
			&item.Name,
			&item.Classification,
			&item.TotalDebit,
			&item.TotalCredit,
			&item.NetBalanceDR,
			&item.NetBalanceCR,
		)
		if err != nil {
			return nil, err
		}

		// Calculate classification-aware balance
		switch item.Classification {
		case "ASSET", "EXPENSE":
			item.Balance = item.NetBalanceDR
		case "LIABILITY", "EQUITY", "REVENUE", "INCOME":
			item.Balance = item.NetBalanceCR
		default:
			item.Balance = item.NetBalanceDR
		}

		list = append(list, item)
	}

	// Add inventory (stock) valuation to the Stock-in-hand account group and all
	// its ancestors. The SQL balance for parent groups (e.g. Current Assets) is
	// computed only from journal entries, so the synthetic stock value must be
	// propagated up the hierarchy manually.
	stockSummary, err := GetCompanyStockSummary(ctx, organizationID)
	if err == nil && len(stockSummary) > 0 {
		var totalStockValue float64
		for _, s := range stockSummary {
			totalStockValue += s.Valuation
		}

		// Build a lookup map for quick parent traversal.
		byID := make(map[string]*GroupBalanceReportItem, len(list))
		for _, item := range list {
			byID[item.AccountGroupID] = item
		}

		// Find Stock-in-hand and walk up to the root, adding stock value at each level.
		for _, item := range list {
			if item.Name == "Stock-in-hand" {
				cur := item
				for cur != nil {
					cur.TotalDebit += totalStockValue
					cur.NetBalanceDR += totalStockValue
					cur.NetBalanceCR -= totalStockValue
					cur.Balance += totalStockValue
					if cur.ParentID == nil {
						break
					}
					cur = byID[*cur.ParentID]
				}
				break
			}
		}
	}

	return list, nil
}

