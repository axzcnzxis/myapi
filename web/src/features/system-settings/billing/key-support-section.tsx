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
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const OPTION_KEYS = {
  enabled: 'key_support_setting.enabled',
  derive: 'key_support_setting.derive_from_subscription',
  fusion: 'key_support_setting.fusion_enabled',
  expose: 'key_support_setting.expose_groups',
  maxGroups: 'key_support_setting.max_groups',
  groupPriority: 'key_support_setting.group_priority',
  balanceEnabled: 'key_support_setting.balance_enabled',
  balanceGroups: 'key_support_setting.balance_groups',
  balanceModels: 'key_support_setting.balance_models',
} as const

export type KeySupportDefaults = {
  enabled: boolean
  derive: boolean
  fusion: boolean
  expose: boolean
  maxGroups: number
  groupPriority: string
  balanceEnabled: boolean
  balanceGroups: string
  balanceModels: string
}

const schema = z.object({
  enabled: z.boolean(),
  derive: z.boolean(),
  fusion: z.boolean(),
  expose: z.boolean(),
  maxGroups: z.coerce.number().int().min(1),
  groupPriority: z.string(),
  balanceEnabled: z.boolean(),
  balanceGroups: z.string(),
  balanceModels: z.string(),
})

type Values = z.infer<typeof schema>

// parseList accepts either a stored JSON array or a comma/newline separated
// draft and returns the trimmed, de-duplicated members.
function parseList(raw: string | undefined): string[] {
  if (!raw) return []
  const trimmed = raw.trim()
  if (trimmed.startsWith('[')) {
    try {
      const parsed = JSON.parse(trimmed)
      if (Array.isArray(parsed)) {
        return Array.from(
          new Set(parsed.map((item) => String(item).trim()).filter(Boolean))
        )
      }
    } catch {
      /* fall through to comma splitting */
    }
  }
  return Array.from(
    new Set(
      trimmed
        .split(/[\n,]/)
        .map((item) => item.trim())
        .filter(Boolean)
    )
  )
}

function toJsonList(raw: string): string {
  return JSON.stringify(parseList(raw))
}

function toDraft(raw: string | undefined): string {
  return parseList(raw).join(', ')
}

export function KeySupportSection({
  defaultValues,
}: {
  defaultValues: KeySupportDefaults
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<Values>({
    resolver: zodResolver(schema) as unknown as Resolver<Values>,
    defaultValues: {
      enabled: defaultValues.enabled,
      derive: defaultValues.derive,
      fusion: defaultValues.fusion,
      expose: defaultValues.expose,
      maxGroups: defaultValues.maxGroups,
      groupPriority: toDraft(defaultValues.groupPriority),
      balanceEnabled: defaultValues.balanceEnabled,
      balanceGroups: toDraft(defaultValues.balanceGroups),
      balanceModels: toDraft(defaultValues.balanceModels),
    },
  })

  const { isDirty, isSubmitting } = form.formState
  const enabled = form.watch('enabled')
  const balanceEnabled = form.watch('balanceEnabled')

  async function onSubmit(values: Values) {
    const updates: Array<{ key: string; value: string }> = []

    if (values.enabled !== defaultValues.enabled) {
      updates.push({ key: OPTION_KEYS.enabled, value: String(values.enabled) })
    }
    if (values.derive !== defaultValues.derive) {
      updates.push({ key: OPTION_KEYS.derive, value: String(values.derive) })
    }
    if (values.fusion !== defaultValues.fusion) {
      updates.push({ key: OPTION_KEYS.fusion, value: String(values.fusion) })
    }
    if (values.expose !== defaultValues.expose) {
      updates.push({ key: OPTION_KEYS.expose, value: String(values.expose) })
    }
    if (values.maxGroups !== defaultValues.maxGroups) {
      updates.push({
        key: OPTION_KEYS.maxGroups,
        value: String(values.maxGroups),
      })
    }
    if (values.groupPriority !== toDraft(defaultValues.groupPriority)) {
      updates.push({
        key: OPTION_KEYS.groupPriority,
        value: toJsonList(values.groupPriority),
      })
    }
    if (values.balanceEnabled !== defaultValues.balanceEnabled) {
      updates.push({
        key: OPTION_KEYS.balanceEnabled,
        value: String(values.balanceEnabled),
      })
    }
    if (values.balanceGroups !== toDraft(defaultValues.balanceGroups)) {
      updates.push({
        key: OPTION_KEYS.balanceGroups,
        value: toJsonList(values.balanceGroups),
      })
    }
    if (values.balanceModels !== toDraft(defaultValues.balanceModels)) {
      updates.push({
        key: OPTION_KEYS.balanceModels,
        value: toJsonList(values.balanceModels),
      })
    }

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }

    form.reset(values)
  }

  return (
    <SettingsSection title={t('Key Support')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending || isSubmitting}
            isSaveDisabled={!isDirty}
            saveLabel='Save key support settings'
          />

          <FormField
            control={form.control}
            name='enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable key support policy')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Derive API key groups from the user subscriptions and restrict which models the balance may pay for'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={updateOption.isPending || isSubmitting}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          {enabled && (
            <>
              <FormField
                control={form.control}
                name='derive'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>
                        {t('Derive groups from subscriptions')}
                      </FormLabel>
                      <FormDescription>
                        {t(
                          'Keys created with the Auto group inherit the groups of the user active subscriptions; a user without a subscription falls back to the default group'
                        )}
                      </FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                        disabled={updateOption.isPending || isSubmitting}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />

              <FormField
                control={form.control}
                name='fusion'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Fuse multiple plans')}</FormLabel>
                      <FormDescription>
                        {t(
                          'When disabled only the highest-priority subscription group is used instead of fusing every subscribed plan into one key'
                        )}
                      </FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                        disabled={updateOption.isPending || isSubmitting}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />

              <FormField
                control={form.control}
                name='expose'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Show actual groups in the key list')}</FormLabel>
                      <FormDescription>
                        {t(
                          'Expand the Auto group of a key into the concrete group set it carries'
                        )}
                      </FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                        disabled={updateOption.isPending || isSubmitting}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />

              <div className='grid gap-6 sm:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='maxGroups'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Maximum derived groups')}</FormLabel>
                      <FormControl>
                        <Input type='number' min={1} {...field} />
                      </FormControl>
                      <FormDescription>
                        {t('Upper bound of groups a single derived key may carry')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='groupPriority'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Group priority')}</FormLabel>
                      <FormControl>
                        <Input
                          placeholder='vip, svip'
                          {...field}
                          value={field.value}
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          'Comma separated, highest priority first. When a model is served by several subscription groups the earlier group wins'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              <FormField
                control={form.control}
                name='balanceEnabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Restrict balance usage')}</FormLabel>
                      <FormDescription>
                        {t(
                          'Limit which groups and models the wallet balance may pay for; leave a list empty for no restriction'
                        )}
                      </FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                        disabled={updateOption.isPending || isSubmitting}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />

              {balanceEnabled && (
                <div className='grid gap-6 sm:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name='balanceGroups'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Groups payable by balance')}</FormLabel>
                        <FormControl>
                          <Input
                            placeholder='default'
                            {...field}
                            value={field.value}
                          />
                        </FormControl>
                        <FormDescription>
                          {t(
                            'Comma separated group names. Empty means every group'
                          )}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name='balanceModels'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Models payable by balance')}</FormLabel>
                        <FormControl>
                          <Input
                            placeholder='gpt-4o-mini, deepseek-chat'
                            {...field}
                            value={field.value}
                          />
                        </FormControl>
                        <FormDescription>
                          {t(
                            'Comma separated model names. Empty means every model'
                          )}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              )}
            </>
          )}
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
