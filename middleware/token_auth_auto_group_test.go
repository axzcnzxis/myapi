package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/axzcnzxis/myapi/common"
	"github.com/axzcnzxis/myapi/constant"
	"github.com/axzcnzxis/myapi/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTokenAuthMiddlewareTest opens an in-memory SQLite database through
// model.InitDB so the shared column names are initialized too, then creates the
// tables TokenAuth and the derived auto groups read.
func setupTokenAuthMiddlewareTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	originalIsMasterNode := common.IsMasterNode
	originalRedisEnabled := common.RedisEnabled
	originalSQLitePath := common.SQLitePath
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()
	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")

	common.IsMasterNode = false
	common.RedisEnabled = false
	common.SQLitePath = fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, os.Setenv("SQL_DSN", "local"))
	require.NoError(t, model.InitDB())
	model.LOG_DB = model.DB
	require.NoError(t, model.DB.AutoMigrate(&model.User{}, &model.Token{}, &model.UserSubscription{}))

	t.Cleanup(func() {
		if sqlDB, err := model.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
		model.DB, model.LOG_DB = originalDB, originalLogDB
		common.IsMasterNode = originalIsMasterNode
		common.RedisEnabled = originalRedisEnabled
		common.SQLitePath = originalSQLitePath
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
		if hadSQLDSN {
			require.NoError(t, os.Setenv("SQL_DSN", originalSQLDSN))
		} else {
			require.NoError(t, os.Unsetenv("SQL_DSN"))
		}
	})
}

func createTokenAuthUser(t *testing.T, username string) *model.User {
	t.Helper()
	user := &model.User{
		Username: username, Password: "password-placeholder", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1,
		AffCode: "middleware-token-" + username,
	}
	require.NoError(t, model.DB.Create(user).Error)
	return user
}

func createTokenAuthKey(t *testing.T, user *model.User, key, group string) *model.Token {
	t.Helper()
	token := &model.Token{
		UserId: user.Id, Key: key, Status: common.TokenStatusEnabled, Name: "token-auth-test",
		ExpiredTime: -1, UnlimitedQuota: true, Group: group,
	}
	require.NoError(t, model.DB.Create(token).Error)
	return token
}

func newTokenAuthRouter() *gin.Engine {
	router := gin.New()
	router.GET("/v1/chat/completions", TokenAuth(), func(c *gin.Context) {
		autoGroups, _ := common.GetContextKey(c, constant.ContextKeyTokenAutoGroups)
		c.JSON(http.StatusOK, gin.H{
			"using_group": common.GetContextKeyString(c, constant.ContextKeyUsingGroup),
			"auto_groups": autoGroups,
		})
	})
	return router
}

func tokenAuthRequest(router *gin.Engine, key string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

// An "auto" key must reach the relay: the group is resolved per request to the
// groups the user may actually use, so the middleware must not refuse it as if
// the user had asked for a group they cannot select. "auto" is never listed in
// UserUsableGroups, so the membership test would reject every auto key.
func TestTokenAuthAllowsAutoGroupWithoutUsableGroupEntry(t *testing.T) {
	setupTokenAuthMiddlewareTest(t)
	user := createTokenAuthUser(t, "token-auth-auto-user")
	key := "tokenauthautokey0000000000000000000000"
	createTokenAuthKey(t, user, key, "auto")

	response := tokenAuthRequest(newTokenAuthRouter(), key)

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var body struct {
		UsingGroup string   `json:"using_group"`
		AutoGroups []string `json:"auto_groups"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "auto", body.UsingGroup)
	assert.Equal(t, []string{"default"}, body.AutoGroups,
		"an auto key without a subscription follows the user's base group")
}

// The exemption is limited to "auto": a concrete group the user may not select
// is still refused.
func TestTokenAuthStillRejectsUnusableConcreteGroup(t *testing.T) {
	setupTokenAuthMiddlewareTest(t)
	user := createTokenAuthUser(t, "token-auth-svip-user")
	key := "tokenauthsvipkey0000000000000000000000"
	createTokenAuthKey(t, user, key, "svip")

	response := tokenAuthRequest(newTokenAuthRouter(), key)

	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), "无权访问 svip 分组")
}
