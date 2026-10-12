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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { handleServerError } from '@/lib/handle-server-error'

import { getWalletSupport, updateWalletSupport } from '../api'
import { fetchGroupNames, fetchModelNames } from '../lib'

export function WalletModelsTab() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [groups, setGroups] = useState<string[]>([])
  const [models, setModels] = useState<string[]>([])
  // Tracks whether the administrator edited the form since the last sync, so a
  // background refetch never silently reverts unsaved edits.
  const [dirty, setDirty] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['key-admin-wallet'],
    queryFn: getWalletSupport,
  })
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

  useEffect(() => {
    const config = data?.data
    if (!config || dirty) return
    setGroups(config.balance_groups ?? [])
    setModels(config.balance_models ?? [])
  }, [data, dirty])

  const updateGroups = (next: string[]) => {
    setDirty(true)
    setGroups(next)
  }
  const updateModels = (next: string[]) => {
    setDirty(true)
    setModels(next)
  }

  const groupSelectOptions = useMemo(
    () => (groupOptions ?? []).map((value) => ({ label: value, value })),
    [groupOptions]
  )
  const modelSelectOptions = useMemo(
    () => (modelNames ?? []).map((value) => ({ label: value, value })),
    [modelNames]
  )

  const saveMutation = useMutation({
    mutationFn: () =>
      updateWalletSupport({
        balance_groups: groups,
        balance_models: models,
      }),
    onSuccess: (result) => {
      if (result.success) {
        toast.success(t('Saved'))
        setDirty(false)
        queryClient.invalidateQueries({ queryKey: ['key-admin-wallet'] })
      } else {
        handleServerError(result, t('Operation failed'))
      }
    },
    onError: (error) => handleServerError(error, t('Operation failed')),
  })

  return (
    <div className='flex max-w-3xl flex-col gap-5'>
      <p className='text-muted-foreground text-sm'>
        {t(
          'Choose which groups and models the wallet balance may pay for. A listed group is also made reachable for "auto" keys, and a listed model is the only model the wallet path offers, so a balance-only account uses exactly what you list. Leave a list empty for no restriction.'
        )}
      </p>

      <div className='space-y-1.5'>
        <Label>{t('Groups the wallet balance may pay for and reach')}</Label>
        <MultiSelect
          options={groupSelectOptions}
          selected={groups}
          onChange={updateGroups}
          placeholder={t('Leave empty to allow all')}
        />
      </div>

      <div className='space-y-1.5'>
        <Label>{t('Models the wallet balance may pay for')}</Label>
        <MultiSelect
          options={modelSelectOptions}
          selected={models}
          onChange={updateModels}
          placeholder={t('Leave empty to allow all')}
        />
      </div>

      <div>
        <Button
          disabled={saveMutation.isPending || isLoading}
          onClick={() => saveMutation.mutate()}
        >
          {saveMutation.isPending ? t('Saving...') : t('Save changes')}
        </Button>
      </div>
    </div>
  )
}
