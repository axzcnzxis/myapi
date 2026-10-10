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

import { looksLikeHtml } from './rich-content-utils'

describe('rich content auto-detection', () => {
  test('detects HTML documents and fragments', () => {
    const htmlSamples = [
      '<div>Hello</div>',
      '  <p class="notice">Hi</p>',
      '<br/>',
      '<img src="/banner.png" alt="banner">',
      '<!DOCTYPE html><html><body>Hi</body></html>',
      '<TABLE><tr><td>1</td></tr></TABLE>',
      '<h1>Title</h1>',
      '<span style="color:red">x</span>',
    ]

    for (const sample of htmlSamples) {
      expect(looksLikeHtml(sample), sample).toBe(true)
    }
  })

  test('keeps Markdown as Markdown', () => {
    const markdownSamples = [
      '',
      '   ',
      '# Title\n\n- a\n- b',
      '**bold** and _italic_',
      'plain text without markup',
      'Hello <br> world',
      '3 < 5 and 7 > 2',
      '<3 the cat',
      '<https://example.com>',
      '<!-- a comment -->\n# Title',
      '[link](https://example.com)',
    ]

    for (const sample of markdownSamples) {
      expect(looksLikeHtml(sample), sample).toBe(false)
    }
  })
})
