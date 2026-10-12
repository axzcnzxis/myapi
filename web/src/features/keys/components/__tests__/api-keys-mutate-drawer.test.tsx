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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, test } from 'vitest'

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { api } = await import('@/lib/api')
const { ApiKeysProvider } = await import('../api-keys-provider')
const { ApiKeysMutateDrawer } = await import('../api-keys-mutate-drawer')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

type ApiMethod = (url: string, data?: unknown) => Promise<{ data: unknown }>
type MockableApi = {
  get: ApiMethod
  post: ApiMethod
  put: ApiMethod
}
type RenderedDrawer = {
  queryClient: InstanceType<typeof QueryClient>
}

const apiClient = api as unknown as MockableApi
const originalGet = apiClient.get
const originalPost = apiClient.post
const originalPut = apiClient.put
let renderedDrawer: RenderedDrawer | null = null

/**
 * The group list returned by the backend when the "auto" group is explicitly
 * listed in the user usable groups.
 */
const USER_GROUPS_WITH_AUTO = {
  auto: { desc: 'Automatic routing', ratio: 'auto' },
  default: { desc: 'Standard access', ratio: 1 },
  vip: { desc: 'Priority access', ratio: 2 },
}

/**
 * The production default: the backend only advertises the groups derived from
 * the user's subscriptions and never lists "auto" itself.
 */
const USER_GROUPS_WITHOUT_AUTO = {
  default: { desc: 'Standard access', ratio: 1 },
  vip: { desc: 'Priority access', ratio: 2 },
}

/** An existing key owned by the signed-in user, used for the update drawer. */
const EXISTING_KEY = {
  id: 1,
  name: 'existing',
  key: 'sk-****',
  status: 1,
  remain_quota: 0,
  used_quota: 0,
  unlimited_quota: true,
  expired_time: -1,
  created_time: 0,
  accessed_time: 0,
  group: 'default',
  auto_groups: [],
  cross_group_retry: false,
  model_limits_enabled: false,
  model_limits: '',
  allow_ips: '',
}

type ApiFixtureOptions = {
  /** Payload of `GET /api/user/self/groups`. */
  userGroups?: Record<string, unknown>
  /** Payload of `GET /api/token/:id` when the update drawer is rendered. */
  existingKey?: typeof EXISTING_KEY
}

function installApiFixtures(
  createdPayloads: Array<Record<string, unknown>>,
  updatedPayloads: Array<Record<string, unknown>>,
  {
    userGroups = USER_GROUPS_WITH_AUTO,
    existingKey = EXISTING_KEY,
  }: ApiFixtureOptions = {}
) {
  apiClient.get = async (url) => {
    switch (url) {
      case '/api/status':
        return { data: { data: { default_use_auto_group: true } } }
      case '/api/user/models':
        return { data: { success: true, data: [] } }
      case '/api/user/self/groups':
        return {
          data: {
            success: true,
            data: userGroups,
          },
        }
      case '/api/token/auto-groups':
        return {
          data: {
            success: true,
            data: { groups: ['vip', 'default'], max_count: 3 },
          },
        }
      case '/api/token/1':
        return { data: { success: true, data: existingKey } }
      default:
        throw new Error(`Unexpected GET ${url}`)
    }
  }
  apiClient.post = async (url, data) => {
    expect(url).toBe('/api/token/')
    expect(data && typeof data === 'object').toBeTruthy()
    createdPayloads.push(data as Record<string, unknown>)
    return { data: { success: true, data: {} } }
  }
  apiClient.put = async (url, data) => {
    expect(url).toBe('/api/token/')
    expect(data && typeof data === 'object').toBeTruthy()
    updatedPayloads.push(data as Record<string, unknown>)
    return { data: { success: true, data: {} } }
  }
}

function primeSharedQueries(
  queryClient: InstanceType<typeof QueryClient>,
  userGroups: Record<string, unknown> = USER_GROUPS_WITH_AUTO
) {
  const freshAt = Date.now() + 60_000
  queryClient.setQueryData(
    ['status'],
    { default_use_auto_group: true },
    { updatedAt: freshAt }
  )
  queryClient.setQueryData(
    ['user-models'],
    { success: true, data: [] },
    { updatedAt: freshAt }
  )
  queryClient.setQueryData(
    ['user-groups'],
    {
      success: true,
      data: userGroups,
    },
    { updatedAt: freshAt }
  )
  queryClient.setQueryData(
    ['token-auto-groups'],
    {
      success: true,
      data: { groups: ['vip', 'default'], max_count: 3 },
    },
    { updatedAt: freshAt }
  )
}

async function renderDrawer(
  currentRow?: typeof EXISTING_KEY,
  userGroups: Record<string, unknown> = USER_GROUPS_WITH_AUTO
): Promise<void> {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  primeSharedQueries(queryClient, userGroups)
  renderedDrawer = { queryClient }

  render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <ApiKeysProvider>
          <ApiKeysMutateDrawer
            open
            onOpenChange={() => undefined}
            currentRow={currentRow}
          />
        </ApiKeysProvider>
      </I18nextProvider>
    </QueryClientProvider>
  )
  await waitFor(
    () => {
      const saveButton = findButton('Save changes', false)
      expect(saveButton).toBeEnabled()
    },
    { timeout: 1500 }
  )
}

function findButton(text: string, required: true): HTMLButtonElement
function findButton(text: string, required: false): HTMLButtonElement | null
function findButton(text: string, required = true): HTMLButtonElement | null {
  const button = screen
    .queryAllByRole<HTMLButtonElement>('button')
    .find((candidate) => candidate.textContent?.includes(text))
  if (required && !button) {
    throw new Error(`Expected button containing "${text}"`)
  }
  return button ?? null
}

function getControlByLabel(
  labelText: 'Name' | 'Quantity' | 'Group'
): HTMLElement
function getControlByLabel(labelText: string): HTMLElement | null
function getControlByLabel(labelText: string): HTMLElement | null {
  const label = [...document.querySelectorAll<HTMLLabelElement>('label')].find(
    (candidate) => candidate.textContent?.trim() === labelText
  )
  if (!label) return null

  return (
    label.control ??
    label
      .closest('[data-slot="form-item"]')
      ?.querySelector<HTMLElement>(
        '[data-slot="form-control"], input, textarea, button[role="combobox"], [role="group"]'
      ) ??
    null
  )
}

function requireControlByLabel(labelText: string): HTMLElement {
  const control = getControlByLabel(labelText)
  if (!control) {
    throw new Error(`Expected control for label "${labelText}"`)
  }
  return control
}

function changeInput(input: HTMLInputElement, value: string): void {
  fireEvent.input(input, { target: { value } })
}

afterEach(() => {
  apiClient.get = originalGet
  apiClient.post = originalPost
  apiClient.put = originalPut
  localStorage.clear()
  if (renderedDrawer) {
    renderedDrawer.queryClient.clear()
    renderedDrawer = null
  }
})

describe('API keys mutate drawer Auto group integration', () => {
  test('creates every key on the derived Auto group without a group selector', async () => {
    const createdPayloads: Array<Record<string, unknown>> = []
    installApiFixtures(createdPayloads, [])
    await renderDrawer()

    // The group selector is removed for creation: a new key always follows the
    // Auto group, whose group set is derived from the user's subscriptions and
    // the wallet allow-list.
    expect(getControlByLabel('Group')).toBe(null)
    expect(document.body.textContent?.includes('Groups this key may use')).toBe(
      true
    )

    changeInput(requireControlByLabel('Name') as HTMLInputElement, 'batch')
    changeInput(requireControlByLabel('Quantity') as HTMLInputElement, '2')
    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(createdPayloads).toHaveLength(2))

    expect(createdPayloads.length).toBe(2)
    expect(createdPayloads[0]?.name).toBe('batch')
    for (const payload of createdPayloads) {
      expect(payload.group).toBe('auto')
      expect(payload.auto_groups).toEqual([])
      expect(payload.cross_group_retry).toBe(true)
    }
  })

  test('keeps the group selector when updating an existing key', async () => {
    const updatedPayloads: Array<Record<string, unknown>> = []
    installApiFixtures([], updatedPayloads)
    await renderDrawer(EXISTING_KEY)

    // Updates still expose the selector so an existing key can be moved to
    // another group.
    const groupControl = requireControlByLabel('Group')
    expect(groupControl.textContent?.includes('default')).toBe(true)

    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(updatedPayloads).toHaveLength(1))
    expect(updatedPayloads[0]?.group).toBe('default')
  })

  test('creates keys on the derived Auto group when the backend omits the auto group', async () => {
    const createdPayloads: Array<Record<string, unknown>> = []
    installApiFixtures(createdPayloads, [], {
      userGroups: USER_GROUPS_WITHOUT_AUTO,
    })
    await renderDrawer(undefined, USER_GROUPS_WITHOUT_AUTO)

    // The default backend group list never contains "auto", yet a new key must
    // still follow the derived Auto group instead of silently falling back to
    // "default".
    expect(getControlByLabel('Group')).toBe(null)

    changeInput(requireControlByLabel('Name') as HTMLInputElement, 'derived')
    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(createdPayloads).toHaveLength(1))

    expect(createdPayloads[0]?.group).toBe('auto')
    expect(createdPayloads[0]?.auto_groups).toEqual([])
    expect(createdPayloads[0]?.cross_group_retry).toBe(true)
  })

  test('keeps an existing Auto key on auto when the backend omits the auto group', async () => {
    const updatedPayloads: Array<Record<string, unknown>> = []
    const autoKey = { ...EXISTING_KEY, group: 'auto' }
    installApiFixtures([], updatedPayloads, {
      userGroups: USER_GROUPS_WITHOUT_AUTO,
      existingKey: autoKey,
    })
    await renderDrawer(autoKey, USER_GROUPS_WITHOUT_AUTO)

    const groupControl = requireControlByLabel('Group')
    expect(groupControl.textContent?.includes('Auto')).toBe(true)

    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(updatedPayloads).toHaveLength(1))
    expect(updatedPayloads[0]?.group).toBe('auto')
  })
})
