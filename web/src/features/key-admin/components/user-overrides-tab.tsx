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
  skipToken,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { MultiSelect } from '@/components/multi-select'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { searchUsers } from '@/features/users/api'
import { handleServerError } from '@/lib/handle-server-error'

import {
  getUserKeySupportGroups,
  getUserWalletOverride,
  updateUserWalletOverride,
} from '../api'
import { fetchGroupNames, fetchModelNames } from '../lib'
import type { AdminUserSummary } from '../types'
import { KeyAdminMutateDrawer } from './key-admin-mutate-drawer'

function toUserSummaries(response: unknown): AdminUserSummary[] {
  const items = (response as { data?: { items?: unknown } })?.data?.items
  if (!Array.isArray(items)) return []
  return items
    .filter(
      (item): item is { id: number; username: string; display_name?: string } =>
        !!item &&
        typeof item === 'object' &&
        typeof (item as { id?: unknown }).id === 'number' &&
        typeof (item as { username?: unknown }).username === 'string'
    )
    .map((item) => ({
      id: item.id,
      username: item.username,
      display_name: item.display_name,
    }))
}

export function UserOverridesTab() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [selectedUser, setSelectedUser] = useState<AdminUserSummary | null>(
    null
  )
  const [drawerOpen, setDrawerOpen] = useState(false)

  const [groupsOverride, setGroupsOverride] = useState(false)
  const [groups, setGroups] = useState<string[]>([])
  const [modelsOverride, setModelsOverride] = useState(false)
  const [models, setModels] = useState<string[]>([])
  // Tracks whether the administrator edited the form since the last sync, so a
  // background refetch never silently reverts unsaved edits.
  const [dirty, setDirty] = useState(false)

  const usersQuery = useQuery({
    queryKey: ['key-admin-users', keyword],
    queryFn: () => searchUsers({ keyword, p: 1, page_size: 20 }),
  })
  const users = useMemo(
    () => toUserSummaries(usersQuery.data),
    [usersQuery.data]
  )

  const { data: groupOptions } = useQuery({
    queryKey: ['key-admin-groups'],
    queryFn: fetchGroupNames,
    staleTime: 60_000,
  })
  const { data: modelNames } = useQuery({
    queryKey: ['key-admin-models'],
    queryFn: fetchModelNames,
    staleTime: 60_000,
  })

  const selectedUserId = selectedUser?.id
  const overrideQuery = useQuery({
    queryKey: ['key-admin-user-wallet', selectedUserId],
    queryFn:
      selectedUserId === undefined
        ? skipToken
        : () => getUserWalletOverride(selectedUserId),
  })
  const derivedGroupsQuery = useQuery({
    queryKey: ['key-admin-user-groups', selectedUserId],
    queryFn:
      selectedUserId === undefined
        ? skipToken
        : () => getUserKeySupportGroups(selectedUserId),
  })

  // Switching to another user starts from that user's stored override, so a
  // pending edit on the previous user must not block the resync.
  useEffect(() => {
    setDirty(false)
  }, [selectedUserId])

  useEffect(() => {
    const override = overrideQuery.data?.data
    if (!override || dirty) return
    setGroupsOverride(override.balance_groups !== null)
    setGroups(override.balance_groups ?? [])
    setModelsOverride(override.balance_models !== null)
    setModels(override.balance_models ?? [])
  }, [overrideQuery.data, dirty])

  const groupSelectOptions = useMemo(
    () => (groupOptions ?? []).map((value) => ({ label: value, value })),
    [groupOptions]
  )
  const modelSelectOptions = useMemo(
    () => (modelNames ?? []).map((value) => ({ label: value, value })),
    [modelNames]
  )

  const saveMutation = useMutation({
    mutationFn: (userId: number) =>
      updateUserWalletOverride(userId, {
        balance_groups: groupsOverride ? groups : null,
        balance_models: modelsOverride ? models : null,
      }),
    onSuccess: (result) => {
      if (result.success) {
        toast.success(t('Saved'))
        queryClient.invalidateQueries({
          queryKey: ['key-admin-user-wallet', selectedUser?.id],
        })
      } else {
        handleServerError(result, t('Operation failed'))
      }
    },
    onError: (error) => handleServerError(error, t('Operation failed')),
  })

  const clearMutation = useMutation({
    mutationFn: (userId: number) =>
      updateUserWalletOverride(userId, {
        balance_groups: null,
        balance_models: null,
      }),
    onSuccess: (result) => {
      if (result.success) {
        toast.success(t('Override cleared'))
        setGroupsOverride(false)
        setGroups([])
        setModelsOverride(false)
        setModels([])
        queryClient.invalidateQueries({
          queryKey: ['key-admin-user-wallet', selectedUser?.id],
        })
      } else {
        handleServerError(result, t('Operation failed'))
      }
    },
    onError: (error) => handleServerError(error, t('Operation failed')),
  })

  const derivedGroups = derivedGroupsQuery.data?.data?.groups ?? []

  return (
    <div className='flex h-full min-h-0 flex-col gap-4 lg:flex-row'>
      <div className='flex w-full flex-col gap-2 lg:max-w-sm'>
        <div className='flex items-center gap-2'>
          <Input
            value={keywordInput}
            placeholder={t('Search users')}
            onChange={(event) => setKeywordInput(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') setKeyword(keywordInput)
            }}
          />
          <Button variant='outline' onClick={() => setKeyword(keywordInput)}>
            {t('Search')}
          </Button>
        </div>
        <div className='max-h-96 overflow-auto rounded-md border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('User')}</TableHead>
                <TableHead className='text-right'>{t('Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.length === 0 && (
                <TableRow>
                  <TableCell
                    colSpan={2}
                    className='text-muted-foreground py-6 text-center'
                  >
                    {usersQuery.isLoading ? t('Loading...') : t('No data')}
                  </TableCell>
                </TableRow>
              )}
              {users.map((user) => (
                <TableRow key={user.id}>
                  <TableCell>
                    <div className='flex flex-col'>
                      <span>{user.display_name || user.username}</span>
                      <span
                        data-table-text='secondary'
                        className='text-muted-foreground'
                      >
                        {user.username} · #{user.id}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className='text-right'>
                    <Button
                      variant={
                        selectedUser?.id === user.id ? 'default' : 'outline'
                      }
                      size='sm'
                      onClick={() => setSelectedUser(user)}
                    >
                      {t('Configure')}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>

      <div className='min-w-0 flex-1'>
        {!selectedUser ? (
          <p className='text-muted-foreground text-sm'>
            {t('Select a user to configure a per-user wallet override.')}
          </p>
        ) : (
          <div className='flex max-w-2xl flex-col gap-4'>
            <div className='flex flex-wrap items-center justify-between gap-2'>
              <h3 className='text-base font-medium'>
                {selectedUser.display_name || selectedUser.username}
                <span className='text-muted-foreground'>
                  {' '}
                  #{selectedUser.id}
                </span>
              </h3>
              <Button
                size='sm'
                variant='outline'
                onClick={() => {
                  setDrawerOpen(true)
                }}
              >
                <Plus />
                {t('Create API Key')}
              </Button>
            </div>

            <div className='space-y-1.5'>
              <label className='flex items-center gap-2 text-sm'>
                <input
                  type='checkbox'
                  checked={groupsOverride}
                  onChange={(event) => setGroupsOverride(event.target.checked)}
                />
                {t('Override the global group list (payable and reachable)')}
              </label>
              <MultiSelect
                options={groupSelectOptions}
                selected={groups}
                onChange={setGroups}
                disabled={!groupsOverride}
                placeholder={t('Leave empty to allow all')}
              />
            </div>

            <div className='space-y-1.5'>
              <label className='flex items-center gap-2 text-sm'>
                <input
                  type='checkbox'
                  checked={modelsOverride}
                  onChange={(event) => setModelsOverride(event.target.checked)}
                />
                {t('Override the global model list (payable)')}
              </label>
              <MultiSelect
                options={modelSelectOptions}
                selected={models}
                onChange={setModels}
                disabled={!modelsOverride}
                placeholder={t('Leave empty to allow all')}
              />
            </div>

            <div className='space-y-1.5'>
              <Label>{t("Groups the user's keys may use")}</Label>
              <div className='flex flex-wrap items-center gap-2'>
                {derivedGroups.length > 0 ? (
                  derivedGroups.map((group) => (
                    <Badge key={group} variant='secondary'>
                      {group}
                    </Badge>
                  ))
                ) : (
                  <span className='text-muted-foreground text-xs'>
                    {t(
                      'No subscription or wallet group, the key falls back to the default group.'
                    )}
                  </span>
                )}
              </div>
            </div>

            <div className='flex flex-wrap gap-2'>
              <Button
                disabled={saveMutation.isPending}
                onClick={() => saveMutation.mutate(selectedUser.id)}
              >
                {saveMutation.isPending ? t('Saving...') : t('Save changes')}
              </Button>
              <Button
                variant='outline'
                disabled={clearMutation.isPending}
                onClick={() => clearMutation.mutate(selectedUser.id)}
              >
                {t('Clear override')}
              </Button>
            </div>
          </div>
        )}
      </div>

      <KeyAdminMutateDrawer
        open={drawerOpen}
        onOpenChange={setDrawerOpen}
        presetUserId={selectedUser?.id}
      />
    </div>
  )
}
