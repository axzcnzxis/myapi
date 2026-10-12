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
import { getUserGroups, getUserModels } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

/** Turn either a group map or a plain list into a deduplicated name list. */
export function toGroupNames(data: unknown): string[] {
  if (!data) return []
  if (Array.isArray(data)) {
    return [
      ...new Set(
        data.filter((item): item is string => typeof item === 'string')
      ),
    ]
  }
  if (typeof data === 'object') {
    return [...new Set(Object.keys(data as Record<string, unknown>))]
  }
  return []
}

/** Group names available to the signed-in administrator (best effort). */
export async function fetchGroupNames(): Promise<string[]> {
  try {
    const res = requireServerSuccess(await getUserGroups())
    return toGroupNames(res?.data)
  } catch {
    return []
  }
}

/** Model names the signed-in administrator may restrict a key to. */
export async function fetchModelNames(): Promise<string[]> {
  try {
    const res = requireServerSuccess(await getUserModels())
    const list = Array.isArray(res?.data) ? res.data : []
    return [...new Set(list)]
  } catch {
    return []
  }
}
