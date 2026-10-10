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

// Announcement lifecycle statuses (mirrors model.AnnouncementStatus*).
export const ANNOUNCEMENT_STATUSES = [
  'draft',
  'scheduled',
  'published',
  'cancelled',
  'expired',
] as const
export type AnnouncementStatus = (typeof ANNOUNCEMENT_STATUSES)[number]

// Delivery modes (公告方式).
export const ANNOUNCEMENT_MODES = ['notice', 'popup', 'both'] as const
export type AnnouncementMode = (typeof ANNOUNCEMENT_MODES)[number]

// Visual tone reused by the bell timeline / dashboard card.
export const ANNOUNCEMENT_TYPES = [
  'default',
  'ongoing',
  'success',
  'warning',
  'error',
] as const
export type AnnouncementType = (typeof ANNOUNCEMENT_TYPES)[number]

export interface Announcement {
  id: number
  title: string
  content: string
  type: AnnouncementType
  mode: AnnouncementMode
  status: AnnouncementStatus
  // Unix seconds (UTC). 0 means "unset".
  publish_at: number
  published_at: number
  expire_at: number
  // 0 = unlimited daily shows.
  daily_limit: number
  // "HH:MM" in Beijing time; empty means all day.
  show_start_time: string
  show_end_time: string
  publisher_id: number
  publisher_name: string
  created_at: number
  updated_at: number
}

export interface AnnouncementFormData {
  id?: number
  title: string
  content: string
  type: AnnouncementType
  mode: AnnouncementMode
  publish_at: number
  expire_at: number
  daily_limit: number
  show_start_time: string
  show_end_time: string
}

export type AnnouncementScope = 'all' | 'current' | 'history'

export interface GetAnnouncementsParams {
  scope?: AnnouncementScope
  status?: AnnouncementStatus | ''
  p?: number
  page_size?: number
}

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export interface GetAnnouncementsResponse {
  success: boolean
  message?: string
  data?: {
    items: Announcement[]
    total: number
    page: number
    page_size: number
  }
}

export type AnnouncementsDialogType = 'create' | 'update' | 'delete' | 'cancel'
