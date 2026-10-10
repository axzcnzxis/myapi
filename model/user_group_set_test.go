package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSplitUserGroups 验证逗号分隔的用户分组集合：按顺序拆分、去空白、去重。
func TestSplitUserGroups(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		{"", []string{}},
		{"   ", []string{}},
		{",", []string{}},
		{"vip", []string{"vip"}},
		{" vip ", []string{"vip"}},
		{"default,vip", []string{"default", "vip"}},
		{"vip,default", []string{"vip", "default"}},
		{" default , vip ", []string{"default", "vip"}},
		{"vip,,svip", []string{"vip", "svip"}},
		{"vip,vip,default", []string{"vip", "default"}},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, SplitUserGroups(tc.raw), "raw=%q", tc.raw)
	}
}

// TestJoinUserGroups 验证集合按顺序拼接为落库形式。
func TestJoinUserGroups(t *testing.T) {
	assert.Equal(t, "", JoinUserGroups(nil))
	assert.Equal(t, "vip", JoinUserGroups([]string{"vip"}))
	assert.Equal(t, "default,vip", JoinUserGroups([]string{"default", "vip"}))
}

// TestNormalizeUserGroups 验证用户分组落库前的归一化：去空白、去重、保持选择顺序。
func TestNormalizeUserGroups(t *testing.T) {
	assert.Equal(t, "", NormalizeUserGroups(""))
	assert.Equal(t, "vip", NormalizeUserGroups(" vip "))
	assert.Equal(t, "default,vip", NormalizeUserGroups(" default , vip "))
	assert.Equal(t, "vip,default", NormalizeUserGroups("vip,vip,default"))
}

// TestPrimaryUserGroup 验证令牌未指定分组时使用集合首个成员，空集合回退 default。
func TestPrimaryUserGroup(t *testing.T) {
	assert.Equal(t, DefaultUserGroup, PrimaryUserGroup(""))
	assert.Equal(t, DefaultUserGroup, PrimaryUserGroup("  ,  "))
	assert.Equal(t, "vip", PrimaryUserGroup("vip"))
	assert.Equal(t, "vip", PrimaryUserGroup("vip,svip"))
	assert.Equal(t, "svip", PrimaryUserGroup(" svip , vip "))
}
