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
import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import { act, cleanup, renderHook, waitFor } from '@testing-library/react'
import type { TFunction } from 'i18next'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { createAppQueryClient } from '@/lib/query-client'

import {
  applySystemUpdate,
  fetchSystemUpdateStatus,
  parseSystemUpdateErrorCode,
  SystemUpdateError,
  type SystemUpdateErrorCode,
  type SystemUpdateStatus,
} from '../self-update'
import {
  getSelfUpdateErrorMessage,
  getSelfUpdateStateLabel,
  getSelfUpdateUnsupportedReason,
  SYSTEM_UPDATE_POLL_INTERVAL,
  SYSTEM_UPDATE_STATUS_QUERY_KEY,
  useSelfUpdate,
} from '../use-self-update'

const status: SystemUpdateStatus = {
  current_version: 'v0.0.10',
  executable: '/opt/myapi/myapi',
  supported: true,
  writable: true,
  container: false,
  reason: '',
  state: 'idle',
  target_version: '',
  error_code: '',
}

// The catalog keys are the English source strings, so echoing the key asserts
// both the lookup and the wording.
const t = ((key: string) => key) as unknown as TFunction

let client: QueryClient

function Wrapper(props: { children: ReactNode }) {
  return (
    <QueryClientProvider client={client}>{props.children}</QueryClientProvider>
  )
}

function respondWithStatus(overrides: Partial<SystemUpdateStatus> = {}): void {
  vi.mocked(api.get).mockResolvedValue({
    data: { success: true, data: { ...status, ...overrides } },
  })
}

function respondWithAcceptance(
  overrides: Partial<SystemUpdateStatus> = {}
): void {
  vi.mocked(api.post).mockResolvedValue({
    data: { success: true, data: { ...status, ...overrides } },
  })
}

/**
 * Let the work a tick scheduled settle. React Query delivers observer updates
 * from a 0-delay task, and the fake clock schedules a 0-delay task created while
 * a tick is running for the next millisecond, so the clock has to move one step
 * before the component can observe the response.
 */
async function flush(): Promise<void> {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1)
  })
}

/** Wait out one poll interval and settle the response it triggers. */
async function pollOnce(): Promise<void> {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(SYSTEM_UPDATE_POLL_INTERVAL)
  })
  await flush()
}

beforeEach(() => {
  client = createAppQueryClient()
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: status },
  })
  vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, data: status },
  })
})

afterEach(() => {
  cleanup()
  client.clear()
  vi.useRealTimers()
})

describe('system update endpoints', () => {
  test('reads the install capability from the status endpoint', async () => {
    respondWithStatus({ supported: false, reason: 'platform' })
    await expect(fetchSystemUpdateStatus()).resolves.toMatchObject({
      supported: false,
      reason: 'platform',
    })
    expect(api.get).toHaveBeenCalledWith('/api/system-update/status', {
      signal: undefined,
      skipErrorHandler: true,
    })
  })

  test('reports a rejection from the status endpoint with its code', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { success: false, code: 'SYSTEM_UPDATE_UNSUPPORTED' },
    })
    const error = await fetchSystemUpdateStatus().catch((e: unknown) => e)
    expect(error).toBeInstanceOf(SystemUpdateError)
    expect((error as SystemUpdateError).code).toBe('SYSTEM_UPDATE_UNSUPPORTED')
  })

  test('rejects an unexpected status payload instead of guessing', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { success: true, data: { state: 'bogus' } },
    })
    await expect(fetchSystemUpdateStatus()).rejects.toThrow(
      'Unexpected system update status'
    )
  })

  test('starts an update and returns the accepted state', async () => {
    respondWithAcceptance({ state: 'downloading', target_version: 'v0.0.11' })
    await expect(applySystemUpdate('v0.0.11')).resolves.toMatchObject({
      state: 'downloading',
      target_version: 'v0.0.11',
    })
    expect(api.post).toHaveBeenCalledWith(
      '/api/system-update/apply',
      { version: 'v0.0.11' },
      { signal: undefined, skipErrorHandler: true }
    )
  })

  test('maps an unknown rejection code to the generic failure', async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { success: false, code: 'SOMETHING_NEW' },
    })
    const error = await applySystemUpdate('v0.0.11').catch((e: unknown) => e)
    expect((error as SystemUpdateError).code).toBe('SYSTEM_UPDATE_FAILED')
    expect(parseSystemUpdateErrorCode(undefined)).toBe('SYSTEM_UPDATE_FAILED')
    expect(parseSystemUpdateErrorCode('SYSTEM_UPDATE_NOT_NEWER')).toBe(
      'SYSTEM_UPDATE_NOT_NEWER'
    )
  })
})

describe('self update messages', () => {
  test.each([
    ['downloading', 'Downloading the update...'],
    ['verifying', 'Verifying the update...'],
    ['installing', 'Installing the update...'],
    ['restarting', 'Restarting the panel...'],
    ['idle', 'Preparing update...'],
    ['failed', 'Preparing update...'],
  ] as const)('labels the %s state', (state, expected) => {
    expect(getSelfUpdateStateLabel(state, t)).toBe(expected)
  })

  test.each([
    ['SYSTEM_UPDATE_UNSUPPORTED', 'This deployment cannot update itself.'],
    ['SYSTEM_UPDATE_INVALID_VERSION', 'That version cannot be installed.'],
    ['SYSTEM_UPDATE_NOT_NEWER', 'The installed version is already up to date.'],
    ['SYSTEM_UPDATE_IN_PROGRESS', 'Another update is already running.'],
    ['SYSTEM_UPDATE_DOWNLOAD_FAILED', 'Failed to download the update.'],
    [
      'SYSTEM_UPDATE_CHECKSUM_FAILED',
      'The downloaded update failed its checksum.',
    ],
    ['SYSTEM_UPDATE_INSTALL_FAILED', 'Failed to install the update.'],
    [
      'SYSTEM_UPDATE_RESTART_FAILED',
      'The panel failed to restart after the update.',
    ],
    ['SYSTEM_UPDATE_FAILED', 'The update failed.'],
  ] satisfies [SystemUpdateErrorCode, string][])(
    'explains %s',
    (code, expected) => {
      expect(getSelfUpdateErrorMessage(code, t)).toBe(expected)
    }
  )

  test.each([
    ['platform', 'Self-update is not available on this platform.'],
    ['executable', 'The panel program could not be located.'],
    [
      'writable',
      'The panel program is not writable, so it cannot be replaced.',
    ],
    ['', 'This deployment cannot update itself.'],
  ] as const)('explains the %s limitation', (reason, expected) => {
    expect(getSelfUpdateUnsupportedReason(reason, t)).toBe(expected)
  })
})

describe('useSelfUpdate', () => {
  test('does not read the status endpoint while disabled', async () => {
    const hook = renderHook(() => useSelfUpdate({ enabled: false }), {
      wrapper: Wrapper,
    })
    await act(async () => undefined)
    expect(api.get).not.toHaveBeenCalled()
    expect(hook.result.current.status).toBeNull()
    expect(hook.result.current.loading).toBe(false)
    expect(hook.result.current.installing).toBe(false)
  })

  test('reports the install capability once the status loads', async () => {
    respondWithStatus({ writable: false, container: true })
    const hook = renderHook(() => useSelfUpdate({ enabled: true }), {
      wrapper: Wrapper,
    })
    await waitFor(() => expect(hook.result.current.status).not.toBeNull())
    expect(hook.result.current.status).toMatchObject({
      writable: false,
      container: true,
    })
    expect(hook.result.current.loading).toBe(false)
    expect(hook.result.current.installing).toBe(false)
  })

  test('polls while the panel is busy and stops once it is idle', async () => {
    vi.useFakeTimers()
    respondWithStatus({ state: 'downloading', target_version: 'v0.0.11' })
    const hook = renderHook(() => useSelfUpdate({ enabled: true }), {
      wrapper: Wrapper,
    })
    await flush()
    expect(api.get).toHaveBeenCalledTimes(1)
    expect(hook.result.current.installing).toBe(true)

    await pollOnce()
    expect(api.get).toHaveBeenCalledTimes(2)

    respondWithStatus({ state: 'idle' })
    await pollOnce()
    expect(api.get).toHaveBeenCalledTimes(3)
    expect(hook.result.current.status?.state).toBe('idle')
    expect(hook.result.current.installing).toBe(false)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(SYSTEM_UPDATE_POLL_INTERVAL * 3)
    })
    expect(api.get).toHaveBeenCalledTimes(3)
  })

  test('follows an update through the restart and reloads the panel', async () => {
    vi.useFakeTimers()
    const reload = vi.fn()
    respondWithAcceptance({ state: 'restarting', target_version: 'v0.0.11' })
    const hook = renderHook(() => useSelfUpdate({ enabled: true, reload }), {
      wrapper: Wrapper,
    })
    await flush()

    act(() => hook.result.current.start('v0.0.11'))
    await flush()
    expect(api.post).toHaveBeenCalledWith(
      '/api/system-update/apply',
      { version: 'v0.0.11' },
      { signal: undefined, skipErrorHandler: true }
    )
    expect(hook.result.current.status?.state).toBe('restarting')
    expect(hook.result.current.awaitingRestart).toBe(true)
    expect(hook.result.current.installing).toBe(true)
    expect(client.getQueryData(SYSTEM_UPDATE_STATUS_QUERY_KEY)).toMatchObject({
      state: 'restarting',
    })

    // The poll keeps failing while the panel replaces its own program, and the
    // last known state is what the dialog shows.
    vi.mocked(api.get).mockRejectedValue(new Error('connection refused'))
    await pollOnce()
    expect(hook.result.current.status?.state).toBe('restarting')
    expect(reload).not.toHaveBeenCalled()

    // The restarted panel answers again and reports the installed version.
    respondWithStatus({ state: 'idle', current_version: 'v0.0.11' })
    await pollOnce()
    expect(hook.result.current.status?.state).toBe('idle')
    expect(reload).toHaveBeenCalledTimes(1)
  })

  test('surfaces the rejection code of an apply request and clears it', async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { success: false, code: 'SYSTEM_UPDATE_IN_PROGRESS' },
    })
    const hook = renderHook(() => useSelfUpdate({ enabled: true }), {
      wrapper: Wrapper,
    })
    await waitFor(() => expect(hook.result.current.status).not.toBeNull())

    act(() => hook.result.current.start('v0.0.11'))
    await waitFor(() =>
      expect(hook.result.current.errorCode).toBe('SYSTEM_UPDATE_IN_PROGRESS')
    )
    expect(hook.result.current.awaitingRestart).toBe(false)

    act(() => hook.result.current.clearError())
    expect(hook.result.current.errorCode).toBeNull()
  })

  test('reports a failure the background job recorded through the polled status', async () => {
    respondWithStatus({
      state: 'failed',
      error_code: 'SYSTEM_UPDATE_INSTALL_FAILED',
    })
    const hook = renderHook(() => useSelfUpdate({ enabled: true }), {
      wrapper: Wrapper,
    })
    await waitFor(() =>
      expect(hook.result.current.errorCode).toBe('SYSTEM_UPDATE_INSTALL_FAILED')
    )
    expect(hook.result.current.installing).toBe(false)
    expect(hook.result.current.status?.state).toBe('failed')
  })

  test('refreshes the status on demand', async () => {
    const hook = renderHook(() => useSelfUpdate({ enabled: true }), {
      wrapper: Wrapper,
    })
    await waitFor(() => expect(hook.result.current.status).not.toBeNull())
    expect(api.get).toHaveBeenCalledTimes(1)
    await act(async () => {
      await hook.result.current.refresh()
    })
    expect(api.get).toHaveBeenCalledTimes(2)
  })
})
