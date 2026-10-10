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
import { z } from 'zod'

import { dateToUnixTimestamp } from '@/lib/time'

import type {
  AnnouncementFormData,
  AnnouncementMode,
  AnnouncementStatus,
  AnnouncementType,
} from './types'

const CLOCK_PATTERN = /^([01]\d|2[0-3]):[0-5]\d$/

// ============================================================================
// Display labels (English strings double as i18n keys)
// ============================================================================

export const ANNOUNCEMENT_STATUS_LABELS: Record<AnnouncementStatus, string> = {
  draft: 'Draft',
  scheduled: 'Scheduled',
  published: 'Published',
  cancelled: 'Cancelled',
  expired: 'Expired',
}

export const ANNOUNCEMENT_MODE_LABELS: Record<AnnouncementMode, string> = {
  notice: 'Notice Bar',
  popup: 'Popup',
  both: 'Both',
}

export const ANNOUNCEMENT_TYPE_LABELS: Record<AnnouncementType, string> = {
  default: 'Default',
  ongoing: 'Ongoing',
  success: 'Success',
  warning: 'Warning',
  error: 'Error',
}

export function getAnnouncementStatusLabel(status: AnnouncementStatus): string {
  return ANNOUNCEMENT_STATUS_LABELS[status] ?? status
}

export function getAnnouncementStatusBadgeVariant(
  status: AnnouncementStatus
): 'default' | 'secondary' | 'destructive' | 'warning' | 'outline' {
  switch (status) {
    case 'published':
      return 'default'
    case 'scheduled':
      return 'secondary'
    case 'expired':
      return 'warning'
    case 'cancelled':
      return 'destructive'
    default:
      return 'outline'
  }
}

// ============================================================================
// Time helpers (all announcement times are shown in Beijing time)
// ============================================================================

const beijingDateTimeFormatter = new Intl.DateTimeFormat('zh-CN', {
  timeZone: 'Asia/Shanghai',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
})

/** Format Unix seconds (UTC) as a Beijing-time "YYYY-MM-DD HH:mm:ss" string. */
export function formatBeijingTime(ts?: number | null): string {
  if (!ts || ts <= 0) return '-'
  return beijingDateTimeFormatter.format(new Date(ts * 1000))
}

const beijingDayFormatter = new Intl.DateTimeFormat('en-CA', {
  timeZone: 'Asia/Shanghai',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
})

/** Beijing calendar day key (YYYY-MM-DD) for a Unix-seconds timestamp. */
export function getBeijingDayKey(ts: number = Math.floor(Date.now() / 1000)) {
  return beijingDayFormatter.format(new Date(ts * 1000))
}

// ============================================================================
// Form schema & transforms
// ============================================================================

export type AnnouncementFormValues = {
  title: string
  content: string
  type: AnnouncementType
  mode: AnnouncementMode
  publish_at?: Date
  expire_at?: Date
  daily_limit: number
  show_start_time: string
  show_end_time: string
}

export const ANNOUNCEMENT_FORM_DEFAULT_VALUES: AnnouncementFormValues = {
  title: '',
  content: '',
  type: 'default',
  mode: 'notice',
  publish_at: undefined,
  expire_at: undefined,
  daily_limit: 0,
  show_start_time: '',
  show_end_time: '',
}

export function getAnnouncementFormSchema(t: (key: string) => string) {
  return z
    .object({
      title: z
        .string()
        .trim()
        .min(1, t('Announcement title is required'))
        .max(200, t('Announcement title cannot exceed 200 characters')),
      content: z
        .string()
        .trim()
        .min(1, t('Announcement content is required'))
        .max(5000, t('Announcement content cannot exceed 5000 characters')),
      type: z.enum(['default', 'ongoing', 'success', 'warning', 'error']),
      mode: z.enum(['notice', 'popup', 'both']),
      publish_at: z.date().optional(),
      expire_at: z.date().optional(),
      daily_limit: z
        .number()
        .int()
        .min(0, t('Daily show limit cannot be negative')),
      show_start_time: z.string(),
      show_end_time: z.string(),
    })
    .superRefine((values, ctx) => {
      const start = values.show_start_time.trim()
      const end = values.show_end_time.trim()
      if (start === '' && end === '') return
      if (start === '' || end === '') {
        ctx.addIssue({
          code: 'custom',
          path: ['show_end_time'],
          message: t('Show window start and end must be filled together'),
        })
        return
      }
      if (!CLOCK_PATTERN.test(start)) {
        ctx.addIssue({
          code: 'custom',
          path: ['show_start_time'],
          message: t('Show window time format must be HH:MM'),
        })
      }
      if (!CLOCK_PATTERN.test(end)) {
        ctx.addIssue({
          code: 'custom',
          path: ['show_end_time'],
          message: t('Show window time format must be HH:MM'),
        })
      }
      if (
        values.expire_at &&
        values.publish_at &&
        values.expire_at.getTime() <= values.publish_at.getTime()
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['expire_at'],
          message: t('Expire time must be later than publish time'),
        })
      }
    })
}

export function transformAnnouncementToFormDefaults(
  announcement: AnnouncementFormData & {
    publish_at: number
    expire_at: number
  }
): AnnouncementFormValues {
  return {
    title: announcement.title,
    content: announcement.content,
    type: announcement.type,
    mode: announcement.mode,
    publish_at: announcement.publish_at
      ? new Date(announcement.publish_at * 1000)
      : undefined,
    expire_at: announcement.expire_at
      ? new Date(announcement.expire_at * 1000)
      : undefined,
    daily_limit: announcement.daily_limit ?? 0,
    show_start_time: announcement.show_start_time ?? '',
    show_end_time: announcement.show_end_time ?? '',
  }
}

export function transformFormValuesToPayload(
  values: AnnouncementFormValues
): AnnouncementFormData {
  return {
    title: values.title.trim(),
    content: values.content.trim(),
    type: values.type,
    mode: values.mode,
    publish_at: values.publish_at ? dateToUnixTimestamp(values.publish_at) : 0,
    expire_at: values.expire_at ? dateToUnixTimestamp(values.expire_at) : 0,
    daily_limit: values.daily_limit ?? 0,
    show_start_time: values.show_start_time.trim(),
    show_end_time: values.show_end_time.trim(),
  }
}

// ============================================================================
// Popup daily-show accounting (client side, keyed by id + Beijing day)
// ============================================================================

const POPUP_COUNT_PREFIX = 'announcement-popup-count'

function getPopupStorageKey(id: number, dayKey = getBeijingDayKey()): string {
  return `${POPUP_COUNT_PREFIX}:${id}:${dayKey}`
}

export function getPopupShownCount(id: number): number {
  if (typeof window === 'undefined') return 0
  const raw = window.localStorage.getItem(getPopupStorageKey(id))
  const parsed = raw ? Number.parseInt(raw, 10) : 0
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0
}

export function incrementPopupShownCount(id: number): void {
  if (typeof window === 'undefined') return
  const next = getPopupShownCount(id) + 1
  window.localStorage.setItem(getPopupStorageKey(id), String(next))
}

/** Whether the popup may still be shown today (daily_limit 0 = unlimited). */
export function canShowPopupToday(announcement: {
  id: number
  daily_limit: number
}): boolean {
  if (!announcement.daily_limit || announcement.daily_limit <= 0) return true
  return getPopupShownCount(announcement.id) < announcement.daily_limit
}
