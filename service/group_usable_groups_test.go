package service

import (
	"testing"

	"github.com/axzcnzxis/myapi/setting"
	"github.com/axzcnzxis/myapi/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetUserUsableGroupsUnionsMembers 验证用户分组集合的可用分组是各成员可用分组的并集：
// 某个成员用 -:vip 移除 vip 时，其他成员仍可用的 vip 不会被一并删除，单成员行为保持不变。
func TestGetUserUsableGroupsUnionsMembers(t *testing.T) {
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP","svip":"SVIP"}`))

	special := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
	originalSpecial := special.ReadAll()
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		special.Clear()
		special.AddAll(originalSpecial)
	})

	// 成员 a 明确移除 vip，成员 b 不做任何调整。
	special.Set("a", map[string]string{"-:vip": ""})

	groups := GetUserUsableGroups("a,b")
	assert.Contains(t, groups, "default")
	assert.Contains(t, groups, "svip")
	assert.Contains(t, groups, "a")
	assert.Contains(t, groups, "b")
	_, hasVip := groups["vip"]
	assert.True(t, hasVip, "并集不应被单个成员的移除抹掉")

	// 单个成员仍保持原有行为：移除生效。
	_, hasVipSingle := GetUserUsableGroups("a")["vip"]
	assert.False(t, hasVipSingle, "单成员仍应保留其移除设置")

	// 空白分组退化为无成员：仍返回全局可用分组。
	assert.Contains(t, GetUserUsableGroups(""), "default")
}
