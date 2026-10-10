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
package model

import "strings"

// DefaultUserGroup is the group a user falls back to when no active
// subscription serves any group.
const DefaultUserGroup = "default"

// SplitUserGroups splits a comma-separated user group set into its ordered,
// trimmed, de-duplicated members. A user may hold several groups at once: an
// administrator can pick them manually, and a subscription adds every group its
// plan declares.
func SplitUserGroups(raw string) []string {
	parts := strings.Split(raw, ",")
	groups := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		group := strings.TrimSpace(part)
		if group == "" {
			continue
		}
		if _, ok := seen[group]; ok {
			continue
		}
		seen[group] = struct{}{}
		groups = append(groups, group)
	}
	return groups
}

// JoinUserGroups canonicalizes an ordered group set into its stored form.
func JoinUserGroups(groups []string) string {
	return strings.Join(groups, ",")
}

// NormalizeUserGroups canonicalizes a user-supplied group value into its stored
// form: comma-separated, trimmed, de-duplicated, selection order preserved.
func NormalizeUserGroups(raw string) string {
	return JoinUserGroups(SplitUserGroups(raw))
}

// PrimaryUserGroup returns the group a request uses when its token does not pin
// one: the first member of the user's group set, or the default group when the
// set is empty. Tokens may still pin any other member of the set explicitly (or
// "auto" to let the group list decide per request).
func PrimaryUserGroup(raw string) string {
	groups := SplitUserGroups(raw)
	if len(groups) == 0 {
		return DefaultUserGroup
	}
	return groups[0]
}
