package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/axzcnzxis/myapi/common"
	"github.com/axzcnzxis/myapi/constant"
	"github.com/axzcnzxis/myapi/model"
	"github.com/axzcnzxis/myapi/setting"
	"github.com/axzcnzxis/myapi/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configureTokenAutoGroupsTest(t *testing.T, maxCount string, autoGroups string) {
	t.Helper()
	originalMax := setting.GetMaxTokenAutoGroups()
	originalAutoGroups := setting.AutoGroups2JsonString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	require.NoError(t, setting.UpdateMaxTokenAutoGroups(maxCount))
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(autoGroups))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1}`))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(stringInt(originalMax)))
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(originalAutoGroups))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
	})
}

func stringInt(value int) string {
	return fmt.Sprintf("%d", value)
}

func setupTokenAutoGroupsControllerTest(t *testing.T) *model.User {
	t.Helper()
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	user := &model.User{
		Id:       101,
		Username: "token-auto-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

// seedActiveSubscription gives the user an active subscription serving the given
// group. A key bound to "auto" derives its groups from exactly these rows.
func seedActiveSubscription(t *testing.T, userId int, upgradeGroup string) {
	t.Helper()
	now := common.GetTimestamp()
	subscription := &model.UserSubscription{
		UserId:       userId,
		PlanId:       1,
		Status:       "active",
		Source:       "admin",
		StartTime:    now - 60,
		EndTime:      now + 3600,
		UpgradeGroup: upgradeGroup,
	}
	require.NoError(t, model.DB.Create(subscription).Error)
}

func baseAutoTokenRequest(name string) map[string]any {
	return map[string]any{
		"name":              name,
		"expired_time":      -1,
		"remain_quota":      0,
		"unlimited_quota":   true,
		"group":             "auto",
		"cross_group_retry": true,
	}
}

func newTokenAutoGroupsAuthenticatedContext(t *testing.T, method string, target string, body any, userID int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	ctx, recorder := newAuthenticatedContext(t, method, target, body, userID)
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	return ctx, recorder
}

// A token bound to "auto" always carries the groups served by the owner's
// current subscriptions, so it must never end up limited to "default" after a
// plan was purchased, and a hand-picked list must not be able to narrow it.
func TestAddTokenAutoGroupsDerivedFromSubscriptions(t *testing.T) {
	tests := []struct {
		name         string
		subscription string
		requestField bool
		requestValue any
		expected     string
	}{
		{name: "no subscription falls back to base group", expected: `["default"]`},
		{name: "subscription group replaces base group", subscription: "vip", expected: `["vip"]`},
		{name: "hand-picked list is ignored", subscription: "vip", requestField: true, requestValue: []string{"default"}, expected: `["vip"]`},
		{name: "empty hand-picked list is ignored", subscription: "vip", requestField: true, requestValue: []string{}, expected: `["vip"]`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configureTokenAutoGroupsTest(t, "5", `["default","vip"]`)
			user := setupTokenAutoGroupsControllerTest(t)
			if test.subscription != "" {
				seedActiveSubscription(t, user.Id, test.subscription)
			}
			request := baseAutoTokenRequest("create-" + test.name)
			if test.requestField {
				request["auto_groups"] = test.requestValue
			}

			ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", request, user.Id)
			AddToken(ctx)

			response := decodeAPIResponse(t, recorder)
			require.True(t, response.Success, response.Message)
			var token model.Token
			require.NoError(t, model.DB.Where("name = ?", request["name"]).First(&token).Error)
			assert.JSONEq(t, test.expected, token.AutoGroups)
			assert.True(t, token.CrossGroupRetry)

			payload, err := common.Marshal(buildMaskedTokenResponse(&token))
			require.NoError(t, err)
			var responseData struct {
				AutoGroups []string `json:"auto_groups"`
			}
			require.NoError(t, common.Unmarshal(payload, &responseData))
			var expectedGroups []string
			require.NoError(t, common.Unmarshal([]byte(test.expected), &expectedGroups))
			assert.Equal(t, expectedGroups, responseData.AutoGroups)
		})
	}
}

// The derived set follows the administrator-configured priority, so the group
// served first is the highest-priority subscription group rather than whichever
// the plan happened to list first.
func TestAddTokenDerivedAutoGroupsFollowConfiguredPriority(t *testing.T) {
	configureTokenAutoGroupsTest(t, "5", `["default","vip"]`)
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP","svip":"SVIP"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1,"svip":1}`))
	user := setupTokenAutoGroupsControllerTest(t)
	seedActiveSubscription(t, user.Id, "svip")
	seedActiveSubscription(t, user.Id, "vip")

	request := baseAutoTokenRequest("priority-order")
	ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", request, user.Id)
	AddToken(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success)

	var token model.Token
	require.NoError(t, model.DB.Where("name = ?", "priority-order").First(&token).Error)
	assert.JSONEq(t, `["vip","svip"]`, token.AutoGroups)
}

// Updating an "auto" key re-derives its groups, so a snapshot stored before a
// plan was bought is corrected immediately; switching away from "auto" clears
// the derived set and disables cross-group retry.
func TestUpdateTokenAutoGroupsRederivedAndClearedForNonAuto(t *testing.T) {
	tests := []struct {
		name          string
		group         string
		expected      string
		expectedRetry bool
	}{
		{name: "auto re-derives from subscriptions", group: "auto", expected: `["vip"]`, expectedRetry: true},
		{name: "non auto clears derived groups", group: "default", expectedRetry: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configureTokenAutoGroupsTest(t, "5", `["default","vip"]`)
			user := setupTokenAutoGroupsControllerTest(t)
			seedActiveSubscription(t, user.Id, "vip")
			token := seedToken(t, model.DB, user.Id, "update-auto", "update-auto-key")
			token.Group = "auto"
			token.CrossGroupRetry = true
			require.NoError(t, token.SetAutoGroups([]string{"default"}))
			require.NoError(t, model.DB.Save(token).Error)

			request := baseAutoTokenRequest("updated-auto")
			request["id"] = token.Id
			request["status"] = common.TokenStatusEnabled
			request["group"] = test.group
			request["auto_groups"] = []string{"default"}
			ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPut, "/api/token/", request, user.Id)
			UpdateToken(ctx)
			response := decodeAPIResponse(t, recorder)
			require.True(t, response.Success, response.Message)

			var updated model.Token
			require.NoError(t, model.DB.First(&updated, token.Id).Error)
			if test.expected == "" {
				assert.Empty(t, updated.AutoGroups)
			} else {
				assert.JSONEq(t, test.expected, updated.AutoGroups)
			}
			assert.Equal(t, test.expectedRetry, updated.CrossGroupRetry)
		})
	}
}

func TestGetTokenAutoGroupsReturnsFullFilteredGlobalOrderAndLimit(t *testing.T) {
	configureTokenAutoGroupsTest(t, "1", `["vip","missing","default"]`)
	user := setupTokenAutoGroupsControllerTest(t)

	ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodGet, "/api/token/auto-groups", nil, user.Id)
	GetTokenAutoGroups(ctx)

	response := decodeAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	var data struct {
		Groups   []string `json:"groups"`
		MaxCount int      `json:"max_count"`
	}
	require.NoError(t, common.Unmarshal(response.Data, &data))
	assert.Equal(t, []string{"vip", "default"}, data.Groups)
	assert.Equal(t, 1, data.MaxCount)
}
