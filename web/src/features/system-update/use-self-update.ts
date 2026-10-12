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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { TFunction } from 'i18next'
import { useCallback, useEffect, useRef, useState } from 'react'

import {
  applySystemUpdate,
  fetchSystemUpdateStatus,
  parseSystemUpdateErrorCode,
  SYSTEM_UPDATE_ACTIVE_STATES,
  SystemUpdateError,
  type SystemUpdateErrorCode,
  type SystemUpdateReason,
  type SystemUpdateState,
} from './self-update'

/** Progress label for the state the server currently reports. */
export function getSelfUpdateStateLabel(
  state: SystemUpdateState,
  t: TFunction
): string {
  switch (state) {
    case 'downloading':
      return t('Downloading the update...')
    case 'verifying':
      return t('Verifying the update...')
    case 'installing':
      return t('Installing the update...')
    case 'restarting':
      return t('Restarting the panel...')
    default:
      return t('Preparing update...')
  }
}

/** Map a stable failure code to operator-facing text. */
export function getSelfUpdateErrorMessage(
  code: SystemUpdateErrorCode,
  t: TFunction
): string {
  switch (code) {
    case 'SYSTEM_UPDATE_UNSUPPORTED':
      return t('This deployment cannot update itself.')
    case 'SYSTEM_UPDATE_INVALID_VERSION':
      return t('That version cannot be installed.')
    case 'SYSTEM_UPDATE_NOT_NEWER':
      return t('The installed version is already up to date.')
    case 'SYSTEM_UPDATE_IN_PROGRESS':
      return t('Another update is already running.')
    case 'SYSTEM_UPDATE_DOWNLOAD_FAILED':
      return t('Failed to download the update.')
    case 'SYSTEM_UPDATE_CHECKSUM_FAILED':
      return t('The downloaded update failed its checksum.')
    case 'SYSTEM_UPDATE_INSTALL_FAILED':
      return t('Failed to install the update.')
    case 'SYSTEM_UPDATE_RESTART_FAILED':
      return t('The panel failed to restart after the update.')
    default:
      return t('The update failed.')
  }
}

/** Explain why this deployment cannot replace its own program. */
export function getSelfUpdateUnsupportedReason(
  reason: SystemUpdateReason,
  t: TFunction
): string {
  switch (reason) {
    case 'platform':
      return t('Self-update is not available on this platform.')
    case 'executable':
      return t('The panel program could not be located.')
    case 'writable':
      return t('The panel program is not writable, so it cannot be replaced.')
    default:
      return t('This deployment cannot update itself.')
  }
}

export const SYSTEM_UPDATE_STATUS_QUERY_KEY = ['system-update-status'] as const

/** Downloading and installing take minutes, so polling stays deliberately slow. */
export const SYSTEM_UPDATE_POLL_INTERVAL = 2_000

/** Reload the dashboard so the replaced panel's frontend bundle is loaded. */
function reloadPanel(): void {
  window.location.reload()
}

/**
 * Drive one panel self-update: start it, follow the server's progress, and
 * reload once the restarted panel answers again.
 *
 * The install outlives the request that starts it, so the dashboard never waits
 * on a long HTTP call — it polls `GET /api/system-update/status` instead. The
 * poll also runs while the panel restarts, where the request fails until the
 * new process is listening, which is how the reload is timed.
 */
export function useSelfUpdate(options: {
  enabled: boolean
  /**
   * Performs the reload once the restarted panel answers again. Overridable so
   * a test can observe the navigation jsdom refuses to carry out.
   */
  reload?: () => void
}) {
  const queryClient = useQueryClient()
  const reload = options.reload ?? reloadPanel
  const reloadRequested = useRef(false)
  const [awaitingRestart, setAwaitingRestart] = useState(false)
  const [errorCode, setErrorCode] = useState<SystemUpdateErrorCode | null>(null)

  const statusQuery = useQuery({
    queryKey: SYSTEM_UPDATE_STATUS_QUERY_KEY,
    queryFn: ({ signal }) => fetchSystemUpdateStatus(signal),
    enabled: options.enabled,
    retry: false,
    staleTime: 0,
    gcTime: 0,
    refetchOnWindowFocus: false,
    // The poll is expected to fail while the panel restarts, so a failed
    // request must never surface a toast.
    meta: { errorToast: false },
    refetchInterval: (query) => {
      // Keep probing through the restart, when every request fails until the
      // replaced process is listening again.
      if (awaitingRestart) return SYSTEM_UPDATE_POLL_INTERVAL
      const state = query.state.data?.state
      return state && SYSTEM_UPDATE_ACTIVE_STATES.has(state)
        ? SYSTEM_UPDATE_POLL_INTERVAL
        : false
    },
  })

  const status = statusQuery.data ?? null

  const mutation = useMutation({
    mutationFn: (version: string) => applySystemUpdate(version),
    onSuccess: (data) => {
      queryClient.setQueryData(SYSTEM_UPDATE_STATUS_QUERY_KEY, data)
      setErrorCode(null)
      if (data.state === 'restarting') setAwaitingRestart(true)
    },
    onError: (error) => {
      setErrorCode(
        error instanceof SystemUpdateError ? error.code : 'SYSTEM_UPDATE_FAILED'
      )
    },
  })

  // A failure the background job records arrives through the polled status, not
  // through the request that started it.
  useEffect(() => {
    if (status?.state === 'restarting') setAwaitingRestart(true)
    if (status?.state === 'failed') {
      setErrorCode(parseSystemUpdateErrorCode(status.error_code))
    }
  }, [status?.state, status?.error_code])

  // Once the restarted panel answers again it reports the installed version in
  // the idle state; reloading then also picks up the new frontend bundle. The
  // reload is requested at most once, so a later status change cannot repeat it.
  useEffect(() => {
    if (!awaitingRestart || reloadRequested.current) return
    if (statusQuery.isSuccess && status?.state === 'idle') {
      reloadRequested.current = true
      reload()
    }
  }, [reload, awaitingRestart, statusQuery.isSuccess, status?.state])

  const start = useCallback(
    (version: string) => {
      setErrorCode(null)
      mutation.mutate(version)
    },
    [mutation]
  )

  return {
    status,
    loading: statusQuery.isLoading,
    installing: status ? SYSTEM_UPDATE_ACTIVE_STATES.has(status.state) : false,
    starting: mutation.isPending,
    awaitingRestart,
    errorCode,
    start,
    clearError: () => setErrorCode(null),
    refresh: () => statusQuery.refetch(),
  }
}
