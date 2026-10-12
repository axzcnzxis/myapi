package controller

import (
	"net/http"
	"strconv"

	"github.com/axzcnzxis/myapi/common"
	"github.com/axzcnzxis/myapi/i18n"
	"github.com/axzcnzxis/myapi/model"
	"github.com/axzcnzxis/myapi/service"
	keysupport "github.com/axzcnzxis/myapi/setting/key_support_setting"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// 管理员跨用户 API Key 管理
// ---------------------------------------------------------------------------

type adminTokenResponse struct {
	*model.Token
	Username   string   `json:"username,omitempty"`
	AutoGroups []string `json:"auto_groups"`
}

func buildAdminTokenResponse(token *model.Token, username string) *adminTokenResponse {
	masked := *token
	masked.Key = token.GetMaskedKey()
	autoGroups, err := token.GetAutoGroups()
	if err != nil || len(autoGroups) == 0 {
		autoGroups = nil
	}
	return &adminTokenResponse{Token: &masked, Username: username, AutoGroups: autoGroups}
}

// AdminAddToken 为指定用户创建 API Key。
func AdminAddToken(c *gin.Context) {
	request := tokenRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	token := request.Token
	targetUserId := token.UserId
	if targetUserId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if _, err := model.GetUserById(targetUserId, false); err != nil {
		common.ApiError(c, err)
		return
	}
	if len(token.Name) > 50 {
		common.ApiErrorI18n(c, i18n.MsgTokenNameTooLong)
		return
	}
	if !token.UnlimitedQuota {
		if token.RemainQuota < 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaNegative)
			return
		}
		if token.RemainQuota > maxTokenQuota() {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaExceedMax, map[string]any{"Max": maxTokenQuota()})
			return
		}
	}

	if token.Group == "auto" {
		groups, err := service.ResolveKeyGroups(targetUserId)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if err := token.SetAutoGroups(groups); err != nil {
			common.ApiError(c, err)
			return
		}
	} else {
		token.CrossGroupRetry = false
		_ = token.SetAutoGroups(nil)
	}

	key, err := common.GenerateKey()
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgTokenGenerateFailed)
		return
	}
	cleanToken := model.Token{
		UserId:             targetUserId,
		Name:               token.Name,
		Key:                key,
		CreatedTime:        common.GetTimestamp(),
		AccessedTime:       common.GetTimestamp(),
		ExpiredTime:        token.ExpiredTime,
		RemainQuota:        token.RemainQuota,
		UnlimitedQuota:     token.UnlimitedQuota,
		ModelLimitsEnabled: token.ModelLimitsEnabled,
		ModelLimits:        token.ModelLimits,
		AllowIps:           token.AllowIps,
		Group:              token.Group,
		CrossGroupRetry:    token.CrossGroupRetry,
		AutoGroups:         token.AutoGroups,
	}
	if err := cleanToken.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	username := ""
	if user, err := model.GetUserById(targetUserId, false); err == nil {
		username = user.Username
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": buildAdminTokenResponse(&cleanToken, username)})
}

// AdminUpdateToken 管理员更新任意用户的 API Key。
func AdminUpdateToken(c *gin.Context) {
	request := tokenRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	token := request.Token
	cleanToken, err := model.GetTokenById(token.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(token.Name) > 50 {
		common.ApiErrorI18n(c, i18n.MsgTokenNameTooLong)
		return
	}
	if !token.UnlimitedQuota {
		if token.RemainQuota < 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaNegative)
			return
		}
		if token.RemainQuota > maxTokenQuota() {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaExceedMax, map[string]any{"Max": maxTokenQuota()})
			return
		}
	}
	if token.Status == common.TokenStatusEnabled {
		if cleanToken.Status == common.TokenStatusExpired && cleanToken.ExpiredTime <= common.GetTimestamp() && cleanToken.ExpiredTime != -1 {
			common.ApiErrorI18n(c, i18n.MsgTokenExpiredCannotEnable)
			return
		}
		if cleanToken.Status == common.TokenStatusExhausted && cleanToken.RemainQuota <= 0 && !cleanToken.UnlimitedQuota {
			common.ApiErrorI18n(c, i18n.MsgTokenExhaustedCannotEable)
			return
		}
	}
	cleanToken.Name = token.Name
	cleanToken.ExpiredTime = token.ExpiredTime
	cleanToken.RemainQuota = token.RemainQuota
	cleanToken.UnlimitedQuota = token.UnlimitedQuota
	cleanToken.ModelLimitsEnabled = token.ModelLimitsEnabled
	cleanToken.ModelLimits = token.ModelLimits
	cleanToken.AllowIps = token.AllowIps
	cleanToken.Group = token.Group
	cleanToken.CrossGroupRetry = token.CrossGroupRetry
	if token.Group != "auto" {
		cleanToken.CrossGroupRetry = false
		_ = cleanToken.SetAutoGroups(nil)
	} else {
		groups, err := service.ResolveKeyGroups(cleanToken.UserId)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if err := cleanToken.SetAutoGroups(groups); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if err := cleanToken.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	username := ""
	if user, err := model.GetUserById(cleanToken.UserId, false); err == nil {
		username = user.Username
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": buildAdminTokenResponse(cleanToken, username)})
}

// ---------------------------------------------------------------------------
// 钱包余额可用模型配置（全局 + 按用户覆盖）
// ---------------------------------------------------------------------------

// AdminGetWalletKeySupport 返回全局钱包可支付配置。
func AdminGetWalletKeySupport(c *gin.Context) {
	s := keysupport.GetSetting()
	common.ApiSuccess(c, gin.H{
		"balance_groups": s.BalanceGroups,
		"balance_models": s.BalanceModels,
	})
}

type walletKeySupportRequest struct {
	BalanceGroups []string `json:"balance_groups"`
	BalanceModels []string `json:"balance_models"`
}

// AdminUpdateWalletKeySupport 更新全局钱包可支付配置（写入 options 表）。
func AdminUpdateWalletKeySupport(c *gin.Context) {
	req := walletKeySupportRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	updates := map[string]string{}
	if req.BalanceGroups != nil {
		updates[keysupport.ConfigName+".balance_groups"] = mustMarshalStringList(req.BalanceGroups)
	}
	if req.BalanceModels != nil {
		updates[keysupport.ConfigName+".balance_models"] = mustMarshalStringList(req.BalanceModels)
	}
	if len(updates) == 0 {
		common.ApiSuccess(c, gin.H{"updated": 0})
		return
	}
	if err := model.UpdateOptionsBulk(updates); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"updated": len(updates)})
}

func mustMarshalStringList(list []string) string {
	if list == nil {
		return "[]"
	}
	data, err := common.Marshal(list)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// AdminGetUserWalletOverride 返回指定用户的钱包可支付覆盖配置。
func AdminGetUserWalletOverride(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	setting := user.GetSetting()
	common.ApiSuccess(c, gin.H{
		"user_id":        userId,
		"username":       user.Username,
		"balance_groups": setting.GetWalletBalanceGroups(),
		"balance_models": setting.GetWalletBalanceModels(),
	})
}

type userWalletOverrideRequest struct {
	BalanceGroups []string `json:"balance_groups"`
	BalanceModels []string `json:"balance_models"`
}

// AdminUpdateUserWalletOverride 设置/清除指定用户的钱包可支付覆盖。
func AdminUpdateUserWalletOverride(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	req := userWalletOverrideRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	setting := user.GetSetting()
	if req.BalanceGroups != nil {
		setting.WalletBalanceGroups = mustMarshalStringList(req.BalanceGroups)
	} else {
		// 未提供则清除覆盖，回退到继承全局。
		setting.WalletBalanceGroups = ""
	}
	if req.BalanceModels != nil {
		setting.WalletBalanceModels = mustMarshalStringList(req.BalanceModels)
	} else {
		setting.WalletBalanceModels = ""
	}
	if err := model.UpdateUserSetting(userId, setting); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"user_id":        userId,
		"balance_groups": setting.GetWalletBalanceGroups(),
		"balance_models": setting.GetWalletBalanceModels(),
	})
}

// AdminGetUserKeySupportGroups 返回指定用户按订阅推导出的分组（管理员预览）。
func AdminGetUserKeySupportGroups(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	groups, err := service.ResolveKeyGroups(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"groups": groups})
}

// GetTokenWalletSupport 返回当前用户钱包可支付的分组/模型（含按用户覆盖）。
func GetTokenWalletSupport(c *gin.Context) {
	userId := c.GetInt("id")
	userSetting, err := model.GetUserSetting(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	global := keysupport.GetSetting()
	groups := userSetting.GetWalletBalanceGroups()
	if groups == nil {
		groups = global.BalanceGroups
	}
	models := userSetting.GetWalletBalanceModels()
	if models == nil {
		models = global.BalanceModels
	}
	common.ApiSuccess(c, gin.H{
		"groups": groups,
		"models": models,
	})
}
