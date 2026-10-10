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
  Announcement,
  AnnouncementFormData,
  ApiResponse,
  GetAnnouncementsParams,
  GetAnnouncementsResponse,
} from './types'

// ============================================================================
// Announcement Management (admin)
// ============================================================================

export async function getAnnouncements(
  params: GetAnnouncementsParams = {}
): Promise<GetAnnouncementsResponse> {
  const { scope = 'all', status = '', p = 1, page_size = 10 } = params
  const query = new URLSearchParams()
  query.set('p', String(p))
  query.set('page_size', String(page_size))
  if (scope && scope !== 'all') query.set('scope', scope)
  if (status) query.set('status', status)
  const res = await api.get(`/api/announcement/?${query.toString()}`)
  return res.data
}

export async function getAnnouncement(
  id: number
): Promise<ApiResponse<Announcement>> {
  const res = await api.get(`/api/announcement/${id}`)
  return res.data
}

export async function createAnnouncement(
  data: AnnouncementFormData
): Promise<ApiResponse<Announcement>> {
  const res = await api.post('/api/announcement/', data)
  return res.data
}

export async function updateAnnouncement(
  data: AnnouncementFormData & { id: number }
): Promise<ApiResponse<Announcement>> {
  const res = await api.put('/api/announcement/', data)
  return res.data
}

export async function deleteAnnouncement(id: number): Promise<ApiResponse> {
  const res = await api.delete(`/api/announcement/${id}`)
  return res.data
}

export async function publishAnnouncement(
  id: number
): Promise<ApiResponse<Announcement>> {
  const res = await api.post(`/api/announcement/${id}/publish`)
  return res.data
}

export async function cancelAnnouncement(
  id: number
): Promise<ApiResponse<Announcement>> {
  const res = await api.post(`/api/announcement/${id}/cancel`)
  return res.data
}

// ============================================================================
// Public endpoints (pre-login home + console popup)
// ============================================================================

// Newest eligible popup announcement, or null when none should be shown.
export async function getPublicPopupAnnouncement(): Promise<
  ApiResponse<Announcement | null>
> {
  const res = await api.get('/api/announcements/popup')
  return res.data
}
