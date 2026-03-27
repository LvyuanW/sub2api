package admin

import (
	"context"
	"sort"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const accountSelectionPageCap = 1000

type AccountSelectionFilters struct {
	Platform    string `json:"platform"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Group       string `json:"group"`
	Search      string `json:"search"`
	PrivacyMode string `json:"privacy_mode"`
}

type AccountSelectionRequest struct {
	AccountIDs []int64                  `json:"account_ids"`
	Filters    *AccountSelectionFilters `json:"filters"`
}

type AccountSelectionPreviewResponse struct {
	Total     int      `json:"total"`
	Platforms []string `json:"platforms"`
	Types     []string `json:"types"`
}

type resolvedAccountSelection struct {
	AccountIDs []int64
	Total      int
	Platforms  []string
	Types      []string
}

func normalizeAccountSelectionFilters(filters AccountSelectionFilters) AccountSelectionFilters {
	filters.Platform = strings.TrimSpace(filters.Platform)
	filters.Type = strings.TrimSpace(filters.Type)
	filters.Status = strings.TrimSpace(filters.Status)
	filters.Group = strings.TrimSpace(filters.Group)
	filters.Search = strings.TrimSpace(filters.Search)
	if len(filters.Search) > 100 {
		filters.Search = filters.Search[:100]
	}
	filters.PrivacyMode = strings.TrimSpace(filters.PrivacyMode)
	return filters
}

func accountSelectionFiltersFromQuery(c *gin.Context) AccountSelectionFilters {
	return normalizeAccountSelectionFilters(AccountSelectionFilters{
		Platform:    c.Query("platform"),
		Type:        c.Query("type"),
		Status:      c.Query("status"),
		Group:       c.Query("group"),
		Search:      c.Query("search"),
		PrivacyMode: c.Query("privacy_mode"),
	})
}

func parseAccountSelectionGroupID(group string) (int64, error) {
	group = strings.TrimSpace(group)
	if group == "" {
		return 0, nil
	}
	if group == accountListGroupUngroupedQueryValue {
		return service.AccountListGroupUngrouped, nil
	}

	groupID, err := strconv.ParseInt(group, 10, 64)
	if err != nil || groupID < 0 {
		return 0, infraerrors.BadRequest("INVALID_GROUP_FILTER", "invalid group filter")
	}
	return groupID, nil
}

func buildResolvedAccountSelection(accounts []service.Account) resolvedAccountSelection {
	result := resolvedAccountSelection{
		AccountIDs: make([]int64, 0, len(accounts)),
		Total:      len(accounts),
	}

	platformSet := make(map[string]struct{}, len(accounts))
	typeSet := make(map[string]struct{}, len(accounts))

	for i := range accounts {
		account := accounts[i]
		if account.ID > 0 {
			result.AccountIDs = append(result.AccountIDs, account.ID)
		}
		if account.Platform != "" {
			platformSet[account.Platform] = struct{}{}
		}
		if account.Type != "" {
			typeSet[account.Type] = struct{}{}
		}
	}

	result.Platforms = make([]string, 0, len(platformSet))
	for platform := range platformSet {
		result.Platforms = append(result.Platforms, platform)
	}
	sort.Strings(result.Platforms)

	result.Types = make([]string, 0, len(typeSet))
	for accountType := range typeSet {
		result.Types = append(result.Types, accountType)
	}
	sort.Strings(result.Types)

	return result
}

func (h *AccountHandler) listAccountsForSelectionFilters(ctx context.Context, filters AccountSelectionFilters) ([]service.Account, error) {
	filters = normalizeAccountSelectionFilters(filters)

	groupID, err := parseAccountSelectionGroupID(filters.Group)
	if err != nil {
		return nil, err
	}

	page := 1
	pageSize := accountSelectionPageCap
	var out []service.Account

	for {
		items, total, err := h.adminService.ListAccounts(
			ctx,
			page,
			pageSize,
			filters.Platform,
			filters.Type,
			filters.Status,
			filters.Search,
			groupID,
			filters.PrivacyMode,
		)
		if err != nil {
			return nil, err
		}

		out = append(out, items...)
		if len(out) >= int(total) || len(items) == 0 {
			break
		}
		page++
	}

	return out, nil
}

func (h *AccountHandler) resolveAccountSelection(ctx context.Context, req AccountSelectionRequest) (resolvedAccountSelection, error) {
	accountIDs := normalizeInt64IDList(req.AccountIDs)

	if len(accountIDs) > 0 && req.Filters != nil {
		return resolvedAccountSelection{}, infraerrors.BadRequest(
			"INVALID_ACCOUNT_SELECTION",
			"exactly one of account_ids or filters is allowed",
		)
	}
	if len(accountIDs) == 0 && req.Filters == nil {
		return resolvedAccountSelection{}, infraerrors.BadRequest(
			"ACCOUNT_SELECTION_REQUIRED",
			"account selection is required",
		)
	}

	if len(accountIDs) > 0 {
		accounts, err := h.adminService.GetAccountsByIDs(ctx, accountIDs)
		if err != nil {
			return resolvedAccountSelection{}, err
		}

		resolved := make([]service.Account, 0, len(accounts))
		for _, account := range accounts {
			if account == nil {
				continue
			}
			resolved = append(resolved, *account)
		}
		return buildResolvedAccountSelection(resolved), nil
	}

	accounts, err := h.listAccountsForSelectionFilters(ctx, *req.Filters)
	if err != nil {
		return resolvedAccountSelection{}, err
	}

	return buildResolvedAccountSelection(accounts), nil
}
