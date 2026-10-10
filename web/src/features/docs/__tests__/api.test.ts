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
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { DOCS_REQUEST_TIMEOUT, fetchDocsMarkdown } from '../api'
import { resolveDocsSource } from '../source'

const source = resolveDocsSource()
const fetchMock = vi.fn<typeof fetch>()

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('documentation fetch', () => {
  test('returns the markdown with repository images resolved', async () => {
    fetchMock.mockResolvedValue(new Response('# Title\n\n![logo](/logo.png)'))

    await expect(
      fetchDocsMarkdown(source, new AbortController().signal)
    ).resolves.toBe(`# Title\n\n![logo](${source.rawBaseUrl}logo.png)`)

    expect(String(fetchMock.mock.calls[0][0])).toBe(source.rawUrl)
    expect(fetchMock.mock.calls[0][1]).toMatchObject({
      cache: 'no-store',
      credentials: 'omit',
    })
  })

  test('reports an HTTP failure together with its status', async () => {
    fetchMock.mockResolvedValue(new Response('', { status: 404 }))

    await expect(
      fetchDocsMarkdown(source, new AbortController().signal)
    ).rejects.toMatchObject({ name: 'DocsError', code: 'http', status: 404 })
  })

  test('reports a network failure', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'))

    await expect(
      fetchDocsMarkdown(source, new AbortController().signal)
    ).rejects.toMatchObject({ name: 'DocsError', code: 'network' })
  })

  test('reports a timeout once the request exceeds its deadline', async () => {
    vi.useFakeTimers()
    fetchMock.mockImplementation(
      (_url, init) =>
        new Promise((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () =>
            reject(new DOMException('Aborted', 'AbortError'))
          )
        })
    )

    const pending = fetchDocsMarkdown(source, new AbortController().signal)
    const assertion = expect(pending).rejects.toMatchObject({
      name: 'DocsError',
      code: 'timeout',
    })
    await vi.advanceTimersByTimeAsync(DOCS_REQUEST_TIMEOUT + 1)
    await assertion
  })

  test('propagates an outer abort instead of reporting a timeout', async () => {
    fetchMock.mockImplementation(
      (_url, init) =>
        new Promise((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () =>
            reject(new DOMException('Aborted', 'AbortError'))
          )
        })
    )

    const controller = new AbortController()
    const pending = fetchDocsMarkdown(source, controller.signal)
    controller.abort()

    await expect(pending).rejects.toBeInstanceOf(DOMException)
  })
})
