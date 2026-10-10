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
import {
  parseSystemVersion,
  selectLatestRelease,
  type SystemRelease,
} from './releases'

export type UpdateCheckErrorCode = 'network' | 'timeout' | 'payload'

export class UpdateCheckError extends Error {
  constructor(public readonly code: UpdateCheckErrorCode) {
    super(code)
    this.name = 'UpdateCheckError'
  }
}

const RELEASES_URL =
  'https://api.github.com/repos/axzcnzxis/myapi/releases?per_page=100'
/** CDN copy of the repository version file, reachable when the API is throttled. */
const VERSION_URL =
  'https://raw.githubusercontent.com/axzcnzxis/myapi/main/VERSION'

const REQUEST_TIMEOUT = 10_000

/**
 * Read the current version from the raw copy of the repository.
 *
 * Unauthenticated requests to the GitHub REST API are limited to a few calls
 * per hour per IP and the dashboard holds no token to spend, so a throttled
 * check would otherwise fail for the visitor. The raw file host serves the same
 * revision without that quota; the version it returns is not a published
 * release, hence the synthesised entry with no notes.
 */
async function fetchReleaseFromVersionFile(
  signal: AbortSignal
): Promise<SystemRelease | null> {
  const response = await fetch(VERSION_URL, {
    cache: 'no-store',
    credentials: 'omit',
    signal,
  })
  if (!response.ok) throw new UpdateCheckError('network')

  const tag = (await response.text()).trim()
  if (!parseSystemVersion(tag)) return null
  return {
    tag_name: tag,
    name: null,
    body: null,
    published_at: null,
    prerelease: false,
  }
}

export async function fetchLatestSystemRelease(
  signal: AbortSignal
): Promise<SystemRelease | null> {
  const controller = new AbortController()
  const cancel = () => controller.abort()
  signal.addEventListener('abort', cancel, { once: true })
  if (signal.aborted) controller.abort()
  const timeout = setTimeout(cancel, REQUEST_TIMEOUT)

  try {
    const response = await fetch(RELEASES_URL, {
      credentials: 'omit',
      headers: { Accept: 'application/vnd.github+json' },
      signal: controller.signal,
    })
    if (response.status === 403 || response.status === 429) {
      return await fetchReleaseFromVersionFile(controller.signal)
    }
    if (!response.ok) throw new UpdateCheckError('network')

    try {
      return selectLatestRelease(await response.json())
    } catch (error) {
      if (controller.signal.aborted) throw error
      throw new UpdateCheckError('payload')
    }
  } catch (error) {
    if (signal.aborted) throw error
    if (controller.signal.aborted) throw new UpdateCheckError('timeout')
    if (error instanceof UpdateCheckError) throw error
    throw new UpdateCheckError('network')
  } finally {
    clearTimeout(timeout)
    signal.removeEventListener('abort', cancel)
  }
}
