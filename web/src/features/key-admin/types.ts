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

// ============================================================================
// Admin cross-user API key management
// ============================================================================

/** A token row as returned by /api/key-admin/tokens (key already masked). */
export interface AdminApiKey {
  id: number
  user_id: number
  username?: string
  name: string
  key: string
  status: number
  remain_quota: number
  used_quota: number
  unlimited_quota: boolean
  expired_time: number
  created_time: number
  accessed_time: number
  group?: string | null
  auto_groups?: string[] | null
  cross_group_retry?: boolean
  model_limits_enabled: boolean
  model_limits?: string | null
  allow_ips?: string | null
}

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

/** Body accepted by POST/PUT /api/key-admin/tokens. */
export interface AdminApiKeyFormData {
  user_id: number
  name: string
  remain_quota: number
  expired_time: number
  unlimited_quota: boolean
  model_limits_enabled: boolean
  model_limits: string
  allow_ips: string
  group: string
  auto_groups: string[]
  cross_group_retry: boolean
}

// ============================================================================
// Wallet balance allow-list (global + per-user override)
// ============================================================================

/**
 * The global wallet allow-list. The list is self-applying: a non-empty
 * `balance_groups` restricts what the wallet may pay for and grants those groups
 * to "auto" keys; an empty list means no restriction and no grant.
 */
export interface WalletSupportConfig {
  balance_groups: string[]
  balance_models: string[]
}

/**
 * A per-user override. `balance_groups`/`balance_models === null` inherit the
 * global list, while a non-null but empty array means "no restriction" for that
 * dimension.
 */
export interface UserWalletOverride {
  user_id: number
  username?: string
  balance_groups: string[] | null
  balance_models: string[] | null
}

export interface UserWalletOverridePayload {
  balance_groups: string[] | null
  balance_models: string[] | null
}

// ============================================================================
// Admin user lookup
// ============================================================================

export interface AdminUserSummary {
  id: number
  username: string
  display_name?: string
  group?: string
  status?: number
}
