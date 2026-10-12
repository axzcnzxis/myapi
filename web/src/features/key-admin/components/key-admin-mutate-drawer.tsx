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
import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { DateTimePicker } from '@/components/datetime-picker'
import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { getCurrencyLabel } from '@/lib/currency'
import {
  getEditableQuotaStep,
  parseQuotaFromDollars,
  quotaUnitsToEditableAmount,
} from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import { createAdminToken, updateAdminToken } from '../api'
import { fetchGroupNames, fetchModelNames } from '../lib'
import type { AdminApiKey } from '../types'

type KeyAdminMutateDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: AdminApiKey | null
  /** Pre-select a user when creating (e.g. from the user overrides tab). */
  presetUserId?: number
  onSaved?: () => void
}

export function KeyAdminMutateDrawer({
  open,
  onOpenChange,
  currentRow,
  presetUserId,
  onSaved,
}: KeyAdminMutateDrawerProps) {
  const { t } = useTranslation()
  const isUpdate = !!currentRow

  const [userId, setUserId] = useState('')
  const [name, setName] = useState('')
  const [group, setGroup] = useState('auto')
  const [unlimitedQuota, setUnlimitedQuota] = useState(false)
  const [quotaAmount, setQuotaAmount] = useState(0)
  const [expiredAt, setExpiredAt] = useState<Date | undefined>(undefined)
  const [modelLimitsEnabled, setModelLimitsEnabled] = useState(false)
  const [modelLimits, setModelLimits] = useState<string[]>([])
  const [allowIps, setAllowIps] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const { data: groups } = useQuery({
    queryKey: ['key-admin-groups'],
    queryFn: fetchGroupNames,
    enabled: open,
    staleTime: 60_000,
  })
  const { data: models } = useQuery({
    queryKey: ['key-admin-models'],
    queryFn: fetchModelNames,
    enabled: open,
    staleTime: 60_000,
  })

  const groupOptions = useMemo(
    () => [...new Set(['auto', ...(groups ?? [])])],
    [groups]
  )
  const modelOptions = useMemo(
    () => (models ?? []).map((model) => ({ label: model, value: model })),
    [models]
  )

  useEffect(() => {
    if (!open) return
    if (currentRow) {
      setUserId(String(currentRow.user_id))
      setName(currentRow.name)
      setGroup(currentRow.group || 'auto')
      setUnlimitedQuota(currentRow.unlimited_quota)
      setQuotaAmount(quotaUnitsToEditableAmount(currentRow.remain_quota))
      setExpiredAt(
        currentRow.expired_time > 0
          ? new Date(currentRow.expired_time * 1000)
          : undefined
      )
      setModelLimitsEnabled(currentRow.model_limits_enabled)
      setModelLimits(
        currentRow.model_limits
          ? currentRow.model_limits.split(',').filter(Boolean)
          : []
      )
      setAllowIps(currentRow.allow_ips ?? '')
    } else {
      setUserId(presetUserId ? String(presetUserId) : '')
      setName('')
      setGroup('auto')
      setUnlimitedQuota(false)
      setQuotaAmount(0)
      setExpiredAt(undefined)
      setModelLimitsEnabled(false)
      setModelLimits([])
      setAllowIps('')
    }
  }, [open, currentRow, presetUserId])

  const handleSubmit = async () => {
    const targetUserId = Number.parseInt(userId, 10)
    if (!isUpdate && (!targetUserId || targetUserId <= 0)) {
      toast.error(t('Please enter a valid user ID'))
      return
    }
    if (!name.trim()) {
      toast.error(t('Please enter a name'))
      return
    }
    const payload = {
      user_id: targetUserId || currentRow?.user_id || 0,
      name: name.trim(),
      remain_quota: unlimitedQuota
        ? 0
        : parseQuotaFromDollars(Number(quotaAmount) || 0),
      expired_time: expiredAt ? Math.floor(expiredAt.getTime() / 1000) : -1,
      unlimited_quota: unlimitedQuota,
      model_limits_enabled: modelLimitsEnabled,
      model_limits: modelLimitsEnabled ? modelLimits.join(',') : '',
      allow_ips: allowIps,
      group,
      auto_groups: [] as string[],
      cross_group_retry: false,
    }
    setSubmitting(true)
    try {
      const result =
        isUpdate && currentRow
          ? await updateAdminToken({ ...payload, id: currentRow.id })
          : await createAdminToken(payload)
      if (result.success) {
        toast.success(isUpdate ? t('API key updated') : t('API key created'))
        onOpenChange(false)
        onSaved?.()
      } else {
        handleServerError(result, t('Operation failed'))
      }
    } catch (error) {
      handleServerError(error, t('Operation failed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex w-full flex-col gap-4 sm:max-w-[560px]'>
        <SheetHeader>
          <SheetTitle>
            {isUpdate ? t('Update API Key') : t('Create API Key')}
          </SheetTitle>
          <SheetDescription>
            {isUpdate
              ? t('Update the API key for this user.')
              : t('Create an API key on behalf of a user.')}
          </SheetDescription>
        </SheetHeader>

        <div className='flex-1 space-y-4 overflow-y-auto px-1'>
          <div className='space-y-1.5'>
            <Label>{t('User ID')}</Label>
            <Input
              value={userId}
              disabled={isUpdate}
              inputMode='numeric'
              placeholder={t('Target user ID')}
              onChange={(event) => setUserId(event.target.value)}
            />
          </div>

          <div className='space-y-1.5'>
            <Label>{t('Name')}</Label>
            <Input
              value={name}
              placeholder={t('Enter a name')}
              onChange={(event) => setName(event.target.value)}
            />
          </div>

          <div className='space-y-1.5'>
            <Label>{t('Group')}</Label>
            <NativeSelect
              className='w-full'
              value={group}
              onChange={(event) => setGroup(event.target.value)}
            >
              {groupOptions.map((option) => (
                <NativeSelectOption key={option} value={option}>
                  {option}
                </NativeSelectOption>
              ))}
            </NativeSelect>
            <p className='text-muted-foreground text-xs'>
              {t(
                'Use "auto" to let the key follow the user subscriptions automatically.'
              )}
            </p>
          </div>

          <div className='flex items-center justify-between gap-3'>
            <Label className='text-sm'>{t('Unlimited Quota')}</Label>
            <Switch
              checked={unlimitedQuota}
              onCheckedChange={setUnlimitedQuota}
            />
          </div>

          {!unlimitedQuota && (
            <div className='space-y-1.5'>
              <Label>
                {t('Quota ({{currency}})', { currency: getCurrencyLabel() })}
              </Label>
              <Input
                type='number'
                step={getEditableQuotaStep()}
                value={quotaAmount}
                onChange={(event) =>
                  setQuotaAmount(Number.parseFloat(event.target.value) || 0)
                }
              />
            </div>
          )}

          <div className='space-y-1.5'>
            <Label>{t('Expiration Time')}</Label>
            <DateTimePicker
              value={expiredAt}
              onChange={setExpiredAt}
              placeholder={t('Never expires')}
            />
          </div>

          <div className='flex items-center justify-between gap-3'>
            <Label className='text-sm'>{t('Model Limits')}</Label>
            <Switch
              checked={modelLimitsEnabled}
              onCheckedChange={setModelLimitsEnabled}
            />
          </div>

          {modelLimitsEnabled && (
            <MultiSelect
              options={modelOptions}
              selected={modelLimits}
              onChange={setModelLimits}
              placeholder={t('Select models (empty for allow all)')}
            />
          )}

          <div className='space-y-1.5'>
            <Label>{t('IP Whitelist (supports CIDR)')}</Label>
            <Textarea
              value={allowIps}
              className='min-h-20 resize-none'
              placeholder={t('One IP per line (empty for no restriction)')}
              onChange={(event) => setAllowIps(event.target.value)}
            />
          </div>
        </div>

        <SheetFooter className='flex-row justify-end gap-2'>
          <SheetClose
            render={<Button variant='outline' className='w-full sm:w-auto' />}
          >
            {t('Close')}
          </SheetClose>
          <Button
            type='button'
            disabled={submitting}
            onClick={handleSubmit}
            className='w-full sm:w-auto'
          >
            {submitting ? t('Saving...') : t('Save changes')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
