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

// Matches content whose first non-whitespace token is an HTML tag or doctype,
// e.g. "<div>", '<p class="x">', "<br/>", "<!doctype html>". Markdown that only
// embeds inline HTML mid-text (or autolinks like <https://example.com>) does
// not match and keeps rendering as Markdown.
const HTML_START_PATTERN =
  /^<(?:!doctype\s+html\b|[a-z][a-z0-9-]*(?:\s[^>]*)?\/?>)/i

/** Heuristic: whether the content should be rendered as HTML rather than Markdown. */
export function looksLikeHtml(content: string): boolean {
  return HTML_START_PATTERN.test(content.trim())
}
