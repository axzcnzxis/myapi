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
import { describe, expect, test } from 'vitest'

import { resolveDocsSource, resolveRepoImages } from '../source'

const DEFAULT_RAW = 'https://raw.githubusercontent.com/axzcnzxis/myapi/main/'

describe('documentation source resolution', () => {
  test('falls back to the repository this build ships from', () => {
    expect(resolveDocsSource()).toEqual({
      rawUrl: `${DEFAULT_RAW}README.md`,
      rawBaseUrl: DEFAULT_RAW,
      linkBaseUrl: 'https://github.com/axzcnzxis/myapi/blob/main/',
      repository: 'axzcnzxis/myapi',
    })
    expect(resolveDocsSource(null)).toEqual(resolveDocsSource())
    expect(resolveDocsSource('   ')).toEqual(resolveDocsSource())
  })

  test('reads a GitHub blob link', () => {
    expect(
      resolveDocsSource(
        'https://github.com/axzcnzxis/myapi/blob/main/docs/README.md'
      )
    ).toEqual({
      rawUrl: `${DEFAULT_RAW}docs/README.md`,
      rawBaseUrl: `${DEFAULT_RAW}docs/`,
      linkBaseUrl: 'https://github.com/axzcnzxis/myapi/blob/main/docs/',
      repository: 'axzcnzxis/myapi',
    })
  })

  test('reads a raw file link', () => {
    const source = resolveDocsSource(
      'https://raw.githubusercontent.com/other/repo/dev/docs/intro.md'
    )
    expect(source).toEqual({
      rawUrl: 'https://raw.githubusercontent.com/other/repo/dev/docs/intro.md',
      rawBaseUrl: 'https://raw.githubusercontent.com/other/repo/dev/docs/',
      linkBaseUrl: 'https://github.com/other/repo/blob/dev/docs/',
      repository: 'other/repo',
    })
  })

  test.each([
    'https://docs.example.com/',
    'https://github.com/axzcnzxis/myapi',
    'https://github.com/axzcnzxis/myapi/releases/tag/v1.0.0',
    'https://github.com/axzcnzxis/myapi/tree/main/docs',
    'not a url',
    'ftp://github.com/axzcnzxis/myapi/blob/main/README.md',
  ])('ignores a link that carries no markdown file: %s', (link) => {
    expect(resolveDocsSource(link)).toEqual(resolveDocsSource())
  })
})

describe('repository-relative images', () => {
  const base = 'https://raw.githubusercontent.com/axzcnzxis/myapi/main/'

  test.each([
    ['![logo](/web/public/logo.png)', `![logo](${base}web/public/logo.png)`],
    ['![a](./docs/a.png)', `![a](${base}docs/a.png)`],
    ['![b](docs/b.png)', `![b](${base}docs/b.png)`],
  ])('points %s at the raw file host', (markdown, expected) => {
    expect(resolveRepoImages(markdown, base)).toBe(expected)
  })

  test('preserves a markdown title', () => {
    expect(resolveRepoImages('![logo](/logo.png "Logo")', base)).toBe(
      `![logo](${base}logo.png "Logo")`
    )
  })

  test('leaves absolute and protocol-relative sources untouched', () => {
    const markdown = [
      '![remote](https://cdn.example.com/logo.png)',
      '![protocol](//cdn.example.com/logo.png)',
      '![data](data:image/png;base64,AAAA)',
    ].join('\n')
    expect(resolveRepoImages(markdown, base)).toBe(markdown)
  })
})
