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

/**
 * Repository rendered by the in-app documentation page.
 *
 * The dashboard never sends the visitor to GitHub: it pulls the markdown file
 * itself and renders it in place. The source is derived from the operator's
 * `docs_link` setting when that points at a GitHub markdown file, and otherwise
 * falls back to the repository this build ships from.
 */
const DEFAULT_DOCS_OWNER = 'axzcnzxis'
const DEFAULT_DOCS_REPOSITORY = 'myapi'
const DEFAULT_DOCS_REF = 'main'
const DEFAULT_DOCS_PATH = 'README.md'

const RAW_HOST = 'raw.githubusercontent.com'
const BLOB_HOSTS = new Set(['github.com', 'www.github.com'])

export type DocsSource = {
  /** Canonical raw URL of the markdown file that is rendered. */
  rawUrl: string
  /** Directory of {@link rawUrl}, used to resolve repository-relative images. */
  rawBaseUrl: string
  /** GitHub blob directory, used to resolve repository-relative links. */
  linkBaseUrl: string
  /** `owner/repository` label shown above the document. */
  repository: string
}

type RepoLocation = {
  owner: string
  repository: string
  ref: string
  path: string
}

function defaultLocation(): RepoLocation {
  return {
    owner: DEFAULT_DOCS_OWNER,
    repository: DEFAULT_DOCS_REPOSITORY,
    ref: DEFAULT_DOCS_REF,
    path: DEFAULT_DOCS_PATH,
  }
}

function splitPath(url: URL): string[] {
  return url.pathname
    .split('/')
    .filter(Boolean)
    .map((segment) => {
      try {
        return decodeURIComponent(segment)
      } catch {
        return segment
      }
    })
}

/**
 * Read owner/repository/ref/path out of a GitHub URL.
 *
 * Only blob and raw file URLs are recognised; a documentation site hosted
 * anywhere else has no markdown file to inline, so the caller keeps the
 * repository this build ships from.
 */
function parseRepoLocation(raw: string | null | undefined): RepoLocation | null {
  const value = raw?.trim()
  if (!value) return null

  let url: URL
  try {
    url = new URL(value)
  } catch {
    return null
  }
  if (url.protocol !== 'http:' && url.protocol !== 'https:') return null

  const host = url.hostname.toLowerCase()
  const segments = splitPath(url)

  if (BLOB_HOSTS.has(host)) {
    // https://github.com/<owner>/<repo>/blob/<ref>/<path>
    if (segments.length < 5 || segments[2] !== 'blob') return null
    const [owner, repository, , ref, ...rest] = segments
    return { owner, repository, ref, path: rest.join('/') }
  }

  if (host === RAW_HOST) {
    // https://raw.githubusercontent.com/<owner>/<repo>/<ref>/<path>
    if (segments.length < 4) return null
    const [owner, repository, ref, ...rest] = segments
    return { owner, repository, ref, path: rest.join('/') }
  }

  return null
}

function withoutFileName(path: string): string {
  const index = path.lastIndexOf('/')
  return index === -1 ? '' : path.slice(0, index + 1)
}

export function resolveDocsSource(docsLink?: string | null): DocsSource {
  const location = parseRepoLocation(docsLink) ?? defaultLocation()
  const path = location.path || DEFAULT_DOCS_PATH
  const { owner, repository, ref } = location
  const rawRoot = `https://${RAW_HOST}/${owner}/${repository}/${ref}/`

  return {
    rawUrl: `${rawRoot}${path}`,
    rawBaseUrl: `${rawRoot}${withoutFileName(path)}`,
    // GitHub orders the segments owner/repository/blob/ref; the raw host omits
    // `blob`, which is why the two roots are not derived from a shared prefix.
    linkBaseUrl: `https://github.com/${owner}/${repository}/blob/${ref}/${withoutFileName(path)}`,
    repository: `${owner}/${repository}`,
  }
}

const ABSOLUTE_URL = /^[a-z][\d+.a-z-]*:/i

const markdownImage = /(!\[[^\]]*\]\()([^)\s]+)((?:\s+"[^"]*")?\))/g

/**
 * Point repository-relative images at the raw file host.
 *
 * A README written for GitHub mixes absolute (`/web/public/logo.png`) and
 * relative image paths; both would 404 against the dashboard origin.
 */
export function resolveRepoImages(markdown: string, rawBaseUrl: string): string {
  return markdown.replace(
    markdownImage,
    (match, prefix: string, source: string, suffix: string) => {
      if (ABSOLUTE_URL.test(source) || source.startsWith('//')) return match
      return `${prefix}${rawBaseUrl}${source.replace(/^\.?\//, '')}${suffix}`
    }
  )
}
