package controller

import (
	"testing"

	"github.com/axzcnzxis/myapi/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNormalizeAndValidateUpgradeGroup 验证套餐升级分组（含融合分组）的规范化与逐组校验：
// 空白归零；合法组合按去重保序规范存储；任一分组不存在即判为非法。
func TestNormalizeAndValidateUpgradeGroup(t *testing.T) {
	previous := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previous))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":2,"svip":4}`))

	cases := []struct {
		raw       string
		wantValue string
		wantValid bool
	}{
		{"", "", true},
		{"   ", "", true},
		{"default", "default", true},
		{"vip", "vip", true},
		{"default,vip", "default,vip", true},
		{" vip , default ", "vip,default", true},
		{"default,vip,svip", "default,vip,svip", true},
		{"vip,vip,default", "vip,default", true},
		{"vip,ghost", "vip,ghost", false},
		{"ghost", "ghost", false},
		{"default,,", "default", true},
	}
	for _, tc := range cases {
		gotValue, gotValid := normalizeAndValidateUpgradeGroup(tc.raw)
		assert.Equal(t, tc.wantValid, gotValid, "raw=%q", tc.raw)
		assert.Equal(t, tc.wantValue, gotValue, "raw=%q", tc.raw)
	}
}
