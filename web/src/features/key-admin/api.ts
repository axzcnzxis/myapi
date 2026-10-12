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
import { api } from '@/lib/api'

import type {
  AdminApiKey,
  AdminApiKeyFormData,
  ApiResponse,
  UserWalletOverride,
  UserWalletOverridePayload,
  WalletSupportConfig,
} from './types'

// ============================================================================
// Cross-user API key management
// ============================================================================

export async function createAdminToken(
  data: AdminApiKeyFormData
): Promise<ApiResponse<AdminApiKey>> {
  const res = await api.post('/api/key-admin/tokens', data)
  return res.data
}

export async function updateAdminToken(
  data: AdminApiKeyFormData & { id: number }
): Promise<ApiResponse<AdminApiKey>> {
  const { id, ...rest } = data
  const res = await api.put(`/api/key-admin/tokens/${id}`, rest)
  return res.data
}

// ============================================================================
// Wallet balance allow-list
// ============================================================================

export async function getWalletSupport(): Promise<
  ApiResponse<WalletSupportConfig>
> {
  const res = await api.get('/api/key-admin/wallet-models')
  return res.data
}

export async function updateWalletSupport(
  data: Partial<WalletSupportConfig>
): Promise<ApiResponse<{ updated: number }>> {
  const res = await api.put('/api/key-admin/wallet-models', data)
  return res.data
}

export async function getUserWalletOverride(
  userId: number
): Promise<ApiResponse<UserWalletOverride>> {
  const res = await api.get(`/api/key-admin/users/${userId}/wallet-models`)
  return res.data
}

export async function updateUserWalletOverride(
  userId: number,
  data: UserWalletOverridePayload
): Promise<ApiResponse<UserWalletOverride>> {
  const res = await api.put(
    `/api/key-admin/users/${userId}/wallet-models`,
    data
  )
  return res.data
}

export async function getUserKeySupportGroups(
  userId: number
): Promise<ApiResponse<{ groups: string[] }>> {
  const res = await api.get(`/api/key-admin/users/${userId}/key-support-groups`)
  return res.data
}
