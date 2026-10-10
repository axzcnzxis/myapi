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
import { resolveRepoImages, type DocsSource } from './source'

export type DocsErrorCode = 'http' | 'network' | 'timeout'

export class DocsError extends Error {
  constructor(
    public readonly code: DocsErrorCode,
    public readonly status?: number
  ) {
    super(code)
    this.name = 'DocsError'
  }
}

export const DOCS_REQUEST_TIMEOUT = 15_000

/**
 * Download the markdown file rendered by the in-app documentation page.
 *
 * The file is served by a CDN and is intentionally revalidated on every pull so
 * that the 「拉取更新」 action always observes the current revision.
 */
export async function fetchDocsMarkdown(
  source: DocsSource,
  signal: AbortSignal
): Promise<string> {
  const controller = new AbortController()
  const cancel = () => controller.abort()
  signal.addEventListener('abort', cancel, { once: true })
  if (signal.aborted) controller.abort()
  const timeout = setTimeout(cancel, DOCS_REQUEST_TIMEOUT)

  try {
    const response = await fetch(source.rawUrl, {
      cache: 'no-store',
      credentials: 'omit',
      signal: controller.signal,
    })
    if (!response.ok) {
      throw new DocsError('http', response.status)
    }
    return resolveRepoImages(await response.text(), source.rawBaseUrl)
  } catch (error) {
    if (signal.aborted) throw error
    if (controller.signal.aborted) throw new DocsError('timeout')
    if (error instanceof DocsError) throw error
    throw new DocsError('network')
  } finally {
    clearTimeout(timeout)
    signal.removeEventListener('abort', cancel)
  }
}
