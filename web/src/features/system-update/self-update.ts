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

import { api } from '@/lib/api'

/**
 * How far the panel's own update has got. The server reports these through
 * `GET /api/system-update/status`; the dashboard polls that endpoint instead of
 * holding one request open, because installing a release takes minutes and a
 * reverse proxy would cut the request off long before it finished.
 */
export const systemUpdateStateSchema = z.enum([
  'idle',
  'downloading',
  'verifying',
  'installing',
  'restarting',
  'failed',
])

export type SystemUpdateState = z.infer<typeof systemUpdateStateSchema>

/** The panel is busy for every state except these two. */
export const SYSTEM_UPDATE_ACTIVE_STATES: ReadonlySet<SystemUpdateState> =
  new Set(['downloading', 'verifying', 'installing', 'restarting'])

/**
 * Stable failure codes the server reports; the dashboard maps each one to its
 * own translation so an internal message never reaches the operator.
 */
export const systemUpdateErrorCodeSchema = z.enum([
  'SYSTEM_UPDATE_UNSUPPORTED',
  'SYSTEM_UPDATE_INVALID_VERSION',
  'SYSTEM_UPDATE_NOT_NEWER',
  'SYSTEM_UPDATE_IN_PROGRESS',
  'SYSTEM_UPDATE_DOWNLOAD_FAILED',
  'SYSTEM_UPDATE_CHECKSUM_FAILED',
  'SYSTEM_UPDATE_INSTALL_FAILED',
  'SYSTEM_UPDATE_RESTART_FAILED',
  'SYSTEM_UPDATE_FAILED',
])

export type SystemUpdateErrorCode = z.infer<typeof systemUpdateErrorCodeSchema>

/** Why this deployment cannot replace its own program, when it cannot. */
export const systemUpdateReasonSchema = z.enum([
  '',
  'platform',
  'executable',
  'writable',
])

export type SystemUpdateReason = z.infer<typeof systemUpdateReasonSchema>

export const systemUpdateStatusSchema = z.object({
  current_version: z.string(),
  executable: z.string(),
  supported: z.boolean(),
  writable: z.boolean(),
  container: z.boolean(),
  reason: systemUpdateReasonSchema,
  state: systemUpdateStateSchema,
  target_version: z.string(),
  error_code: z.string(),
})

export type SystemUpdateStatus = z.infer<typeof systemUpdateStatusSchema>

/** A rejection the server reported for `POST /apply`, carrying its stable code. */
export class SystemUpdateError extends Error {
  constructor(readonly code: SystemUpdateErrorCode) {
    super(code)
    this.name = 'SystemUpdateError'
  }
}

/** Coerce an arbitrary code to the known set, falling back to the generic one. */
export function parseSystemUpdateErrorCode(
  value: unknown
): SystemUpdateErrorCode {
  const parsed = systemUpdateErrorCodeSchema.safeParse(value)
  return parsed.success ? parsed.data : 'SYSTEM_UPDATE_FAILED'
}

const envelopeSchema = z.object({
  success: z.boolean(),
  code: z.string().optional(),
  data: z.unknown().optional(),
})

function parseEnvelope(payload: unknown): z.infer<typeof envelopeSchema> {
  const parsed = envelopeSchema.safeParse(payload)
  if (!parsed.success) throw new Error('Unexpected system update response')
  return parsed.data
}

function parseStatus(payload: unknown): SystemUpdateStatus {
  const parsed = systemUpdateStatusSchema.safeParse(payload)
  if (!parsed.success) throw new Error('Unexpected system update status')
  return parsed.data
}

/** Read whether this deployment can update itself and how far it has got. */
export async function fetchSystemUpdateStatus(
  signal?: AbortSignal
): Promise<SystemUpdateStatus> {
  const response = await api.get('/api/system-update/status', {
    signal,
    skipErrorHandler: true,
  })
  const envelope = parseEnvelope(response.data)
  if (!envelope.success) {
    throw new SystemUpdateError(parseSystemUpdateErrorCode(envelope.code))
  }
  return parseStatus(envelope.data)
}

/**
 * Start installing a published release over the running program. The server
 * answers as soon as the attempt is accepted and reports progress through the
 * status endpoint; a rejection is returned synchronously with its code.
 */
export async function applySystemUpdate(
  version: string,
  signal?: AbortSignal
): Promise<SystemUpdateStatus> {
  const response = await api.post(
    '/api/system-update/apply',
    { version },
    { signal, skipErrorHandler: true }
  )
  const envelope = parseEnvelope(response.data)
  if (!envelope.success) {
    throw new SystemUpdateError(parseSystemUpdateErrorCode(envelope.code))
  }
  return parseStatus(envelope.data)
}
