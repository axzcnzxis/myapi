/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package key_support_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// setWalletSetting installs a wallet allow-list and returns the restore func.
func setWalletSetting(groups, models []string) func() {
	original := keySupportSetting
	keySupportSetting.BalanceGroups = groups
	keySupportSetting.BalanceModels = models
	return func() { keySupportSetting = original }
}

func withWalletSetting(t *testing.T, groups, models []string) {
	t.Helper()
	t.Cleanup(setWalletSetting(groups, models))
}

// An empty allow-list is the "off" state: the wallet may pay for anything.
func TestCheckWalletAllowedEmptyListIsUnrestricted(t *testing.T) {
	withWalletSetting(t, nil, nil)

	assert.NoError(t, CheckWalletAllowed("vip", "claude-3"))
}

func TestCheckWalletAllowedGlobalList(t *testing.T) {
	withWalletSetting(t, []string{"default"}, []string{"gpt-4"})

	assert.NoError(t, CheckWalletAllowed("default", "gpt-4"))
	assert.Error(t, CheckWalletAllowed("vip", "gpt-4"))
	assert.Error(t, CheckWalletAllowed("default", "claude-3"))
}

func TestCheckWalletAllowedForUserInheritsGlobal(t *testing.T) {
	withWalletSetting(t, []string{"default"}, []string{"gpt-4"})

	// No per-user override at all: the global policy applies unchanged.
	assert.NoError(t, CheckWalletAllowedForUser(nil, nil, "default", "gpt-4"))
	assert.Error(t, CheckWalletAllowedForUser(nil, nil, "vip", "gpt-4"))
	assert.Error(t, CheckWalletAllowedForUser(nil, nil, "default", "claude-3"))
}

// WalletGroupsFor is the single source of truth for the wallet allow-list: a nil
// result means "unrestricted, grants nothing", a non-nil result is both the set
// the wallet may pay for and the set the allow-list makes reachable.
func TestWalletGroupsForEmptyListGrantsNothing(t *testing.T) {
	withWalletSetting(t, nil, nil)
	assert.Nil(t, WalletGroupsFor(nil), "an empty list grants nothing")

	withWalletSetting(t, []string{}, nil)
	assert.Nil(t, WalletGroupsFor(nil), "an empty list grants nothing")

	withWalletSetting(t, []string{"default", "svip"}, nil)
	assert.Equal(t, []string{"default", "svip"}, WalletGroupsFor(nil))
}

func TestWalletGroupsForNormalizesAndOverrides(t *testing.T) {
	withWalletSetting(t, []string{"default"}, nil)

	// A non-nil override replaces the global list, nil inherits it.
	assert.Equal(t, []string{"vip"}, WalletGroupsFor([]string{"vip"}))
	assert.Equal(t, []string{"default"}, WalletGroupsFor(nil))
	// A non-nil but empty override means "no restriction", hence no grant.
	assert.Nil(t, WalletGroupsFor([]string{}))
	// Blank entries are dropped and duplicates collapsed, case-insensitively.
	assert.Equal(t, []string{"vip", "svip"}, WalletGroupsFor([]string{" vip ", "VIP", "svip", "  "}))
	assert.Nil(t, WalletGroupsFor([]string{" ", ""}), "a list of blanks grants nothing")
}

// WalletModelsFor mirrors WalletGroupsFor: a nil result means "unrestricted,
// advertises everything the key can reach", a non-nil result is the exact
// catalogue the wallet path advertises.
func TestWalletModelsForEmptyListIsUnrestricted(t *testing.T) {
	withWalletSetting(t, nil, nil)
	assert.Nil(t, WalletModelsFor(nil), "an empty list restricts nothing")

	withWalletSetting(t, nil, []string{})
	assert.Nil(t, WalletModelsFor(nil), "an empty list restricts nothing")

	withWalletSetting(t, nil, []string{"gpt-4", "claude-3"})
	assert.Equal(t, []string{"gpt-4", "claude-3"}, WalletModelsFor(nil))
}

func TestWalletModelsForNormalizesAndOverrides(t *testing.T) {
	withWalletSetting(t, nil, []string{"gpt-4"})

	// A non-nil override replaces the global list, nil inherits it.
	assert.Equal(t, []string{"claude-3"}, WalletModelsFor([]string{"claude-3"}))
	assert.Equal(t, []string{"gpt-4"}, WalletModelsFor(nil))
	// A non-nil but empty override means "no restriction".
	assert.Nil(t, WalletModelsFor([]string{}))
	// Blank entries are dropped and duplicates collapsed, case-insensitively.
	assert.Equal(t, []string{"gpt-4", "claude-3"}, WalletModelsFor([]string{" gpt-4 ", "GPT-4", "claude-3", "  "}))
	assert.Nil(t, WalletModelsFor([]string{" ", ""}), "a list of blanks restricts nothing")
}

func TestContainsGroupIgnoresCaseAndWhitespace(t *testing.T) {
	assert.True(t, ContainsGroup([]string{" vip ", "SVIP"}, "svip"))
	assert.True(t, ContainsGroup([]string{"vip"}, " VIP "))
	assert.False(t, ContainsGroup([]string{"vip"}, "default"))
	assert.False(t, ContainsGroup(nil, "vip"))
	assert.False(t, ContainsGroup([]string{"vip"}, ""))
}

func TestCheckWalletAllowedForUserOverridesLists(t *testing.T) {
	withWalletSetting(t, []string{"default"}, []string{"gpt-4"})

	// A non-nil list replaces the global one, a nil list keeps inheriting it.
	assert.NoError(t, CheckWalletAllowedForUser([]string{"vip"}, nil, "vip", "gpt-4"))
	assert.Error(t, CheckWalletAllowedForUser([]string{"vip"}, nil, "default", "gpt-4"))
	assert.Error(t, CheckWalletAllowedForUser([]string{"vip"}, nil, "vip", "claude-3"))

	// A non-nil but empty list means "no restriction" for that dimension only.
	assert.NoError(t, CheckWalletAllowedForUser([]string{}, nil, "default", "gpt-4"))
	assert.Error(t, CheckWalletAllowedForUser([]string{}, nil, "default", "claude-3"))
}
