package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type selectionListCall struct {
	page        int
	pageSize    int
	platform    string
	accountType string
	status      string
	search      string
	groupID     int64
	privacyMode string
}

type selectionAwareAdminService struct {
	*stubAdminService
	listCalls           []selectionListCall
	lastBulkUpdateInput *service.BulkUpdateAccountsInput
}

func newSelectionAwareAdminService(accounts []service.Account) *selectionAwareAdminService {
	base := newStubAdminService()
	base.accounts = append([]service.Account(nil), accounts...)
	return &selectionAwareAdminService{stubAdminService: base}
}

func (s *selectionAwareAdminService) ListAccounts(
	ctx context.Context,
	page, pageSize int,
	platform, accountType, status, search string,
	groupID int64,
	privacyMode, sortBy, sortOrder string,
) ([]service.Account, int64, error) {
	s.listCalls = append(s.listCalls, selectionListCall{
		page:        page,
		pageSize:    pageSize,
		platform:    platform,
		accountType: accountType,
		status:      status,
		search:      search,
		groupID:     groupID,
		privacyMode: privacyMode,
	})

	_ = ctx
	_ = sortBy
	_ = sortOrder

	filtered := make([]service.Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		if !matchesSelectionFilters(account, platform, accountType, status, search, groupID, privacyMode) {
			continue
		}
		filtered = append(filtered, account)
	}

	total := int64(len(filtered))
	if pageSize <= 0 {
		return filtered, total, nil
	}

	start := (page - 1) * pageSize
	if start >= len(filtered) {
		return []service.Account{}, total, nil
	}

	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	return append([]service.Account(nil), filtered[start:end]...), total, nil
}

func (s *selectionAwareAdminService) BulkUpdateAccounts(
	ctx context.Context,
	input *service.BulkUpdateAccountsInput,
) (*service.BulkUpdateAccountsResult, error) {
	copied := *input
	copied.AccountIDs = append([]int64(nil), input.AccountIDs...)
	if input.GroupIDs != nil {
		groupIDs := append([]int64(nil), (*input.GroupIDs)...)
		copied.GroupIDs = &groupIDs
	}
	s.lastBulkUpdateInput = &copied
	return s.stubAdminService.BulkUpdateAccounts(ctx, input)
}

func matchesSelectionFilters(
	account service.Account,
	platform, accountType, status, search string,
	groupID int64,
	privacyMode string,
) bool {
	if platform != "" && account.Platform != platform {
		return false
	}
	if accountType != "" && account.Type != accountType {
		return false
	}
	if status != "" && account.Status != status {
		return false
	}

	search = strings.TrimSpace(strings.ToLower(search))
	if search != "" && !strings.Contains(strings.ToLower(account.Name), search) {
		return false
	}

	if groupID == service.AccountListGroupUngrouped {
		if len(account.GroupIDs) > 0 {
			return false
		}
	} else if groupID > 0 && !accountHasGroup(account, groupID) {
		return false
	}

	if privacyMode != "" && selectionPrivacyMode(account) != privacyMode {
		return false
	}

	return true
}

func accountHasGroup(account service.Account, groupID int64) bool {
	for _, id := range account.GroupIDs {
		if id == groupID {
			return true
		}
	}
	return false
}

func selectionPrivacyMode(account service.Account) string {
	if account.Extra == nil {
		return ""
	}
	value, _ := account.Extra["privacy_mode"].(string)
	return strings.TrimSpace(value)
}

func setupAccountSelectionRouter(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	accountHandler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/selection-preview", accountHandler.PreviewSelection)
	router.POST("/api/v1/admin/accounts/bulk-update", accountHandler.BulkUpdate)
	return router
}

func TestPreviewSelectionWithFiltersReturnsMetadata(t *testing.T) {
	adminSvc := newSelectionAwareAdminService([]service.Account{
		{
			ID:       11,
			Name:     "Alpha OpenAI OAuth",
			Platform: "openai",
			Type:     "oauth",
			Status:   "active",
			Extra:    map[string]any{"privacy_mode": "masked"},
		},
		{
			ID:       12,
			Name:     "Alpha OpenAI APIKey",
			Platform: "openai",
			Type:     "apikey",
			Status:   "active",
			GroupIDs: []int64{7},
			Extra:    map[string]any{"privacy_mode": "masked"},
		},
		{
			ID:       13,
			Name:     "Alpha OpenAI OAuth Inactive",
			Platform: "openai",
			Type:     "oauth",
			Status:   "inactive",
			Extra:    map[string]any{"privacy_mode": "masked"},
		},
		{
			ID:       14,
			Name:     "Alpha Anthropic OAuth",
			Platform: "anthropic",
			Type:     "oauth",
			Status:   "active",
			Extra:    map[string]any{"privacy_mode": "masked"},
		},
	})
	router := setupAccountSelectionRouter(adminSvc)

	body, err := json.Marshal(map[string]any{
		"filters": map[string]any{
			"platform":     "openai",
			"type":         "oauth",
			"status":       "active",
			"group":        "ungrouped",
			"search":       "alpha",
			"privacy_mode": "masked",
		},
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/selection-preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int                             `json:"code"`
		Data AccountSelectionPreviewResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 1, resp.Data.Total)
	require.Equal(t, []string{"openai"}, resp.Data.Platforms)
	require.Equal(t, []string{"oauth"}, resp.Data.Types)
	require.Len(t, adminSvc.listCalls, 1)
	require.Equal(t, selectionListCall{
		page:        1,
		pageSize:    accountSelectionPageCap,
		platform:    "openai",
		accountType: "oauth",
		status:      "active",
		search:      "alpha",
		groupID:     service.AccountListGroupUngrouped,
		privacyMode: "masked",
	}, adminSvc.listCalls[0])
}

func TestBulkUpdateWithFiltersResolvesMatchingAccountsAcrossPages(t *testing.T) {
	accounts := make([]service.Account, 0, 1004)
	for i := 1; i <= 1002; i++ {
		accounts = append(accounts, service.Account{
			ID:       int64(i),
			Name:     "Match OpenAI OAuth",
			Platform: "openai",
			Type:     "oauth",
			Status:   "active",
			GroupIDs: []int64{88},
			Extra:    map[string]any{"privacy_mode": "masked"},
		})
	}
	accounts = append(accounts,
		service.Account{
			ID:       2001,
			Name:     "Match OpenAI OAuth Other Group",
			Platform: "openai",
			Type:     "oauth",
			Status:   "active",
			GroupIDs: []int64{99},
			Extra:    map[string]any{"privacy_mode": "masked"},
		},
		service.Account{
			ID:       2002,
			Name:     "Match Anthropic OAuth",
			Platform: "anthropic",
			Type:     "oauth",
			Status:   "active",
			GroupIDs: []int64{88},
			Extra:    map[string]any{"privacy_mode": "masked"},
		},
	)

	adminSvc := newSelectionAwareAdminService(accounts)
	router := setupAccountSelectionRouter(adminSvc)

	body, err := json.Marshal(map[string]any{
		"filters": map[string]any{
			"platform":     "openai",
			"type":         "oauth",
			"status":       "active",
			"group":        "88",
			"search":       "match",
			"privacy_mode": "masked",
		},
		"schedulable": true,
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/bulk-update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int                              `json:"code"`
		Data service.BulkUpdateAccountsResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 1002, resp.Data.Success)
	require.Equal(t, 0, resp.Data.Failed)

	require.NotNil(t, adminSvc.lastBulkUpdateInput)
	require.Len(t, adminSvc.lastBulkUpdateInput.AccountIDs, 1002)
	require.Equal(t, int64(1), adminSvc.lastBulkUpdateInput.AccountIDs[0])
	require.Equal(t, int64(1002), adminSvc.lastBulkUpdateInput.AccountIDs[len(adminSvc.lastBulkUpdateInput.AccountIDs)-1])
	require.NotContains(t, adminSvc.lastBulkUpdateInput.AccountIDs, int64(2001))
	require.NotContains(t, adminSvc.lastBulkUpdateInput.AccountIDs, int64(2002))
	require.NotNil(t, adminSvc.lastBulkUpdateInput.Schedulable)
	require.True(t, *adminSvc.lastBulkUpdateInput.Schedulable)
	require.Len(t, adminSvc.listCalls, 2)
	require.Equal(t, 1, adminSvc.listCalls[0].page)
	require.Equal(t, accountSelectionPageCap, adminSvc.listCalls[0].pageSize)
	require.Equal(t, 2, adminSvc.listCalls[1].page)
	require.Equal(t, accountSelectionPageCap, adminSvc.listCalls[1].pageSize)
}
