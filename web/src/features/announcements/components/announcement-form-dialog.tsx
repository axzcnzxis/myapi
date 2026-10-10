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
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { DateTimePicker } from '@/components/datetime-picker'
import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { handleServerError } from '@/lib/handle-server-error'

import { createAnnouncement, updateAnnouncement } from '../api'
import {
  ANNOUNCEMENT_FORM_DEFAULT_VALUES,
  ANNOUNCEMENT_MODE_LABELS,
  ANNOUNCEMENT_TYPE_LABELS,
  getAnnouncementFormSchema,
  transformAnnouncementToFormDefaults,
  transformFormValuesToPayload,
  type AnnouncementFormValues,
} from '../lib'
import type { Announcement } from '../types'

interface AnnouncementFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  announcement?: Announcement | null
  onSaved: () => void
}

export function AnnouncementFormDialog({
  open,
  onOpenChange,
  announcement,
  onSaved,
}: AnnouncementFormDialogProps) {
  const { t } = useTranslation()
  const isUpdate = !!announcement
  const [isSubmitting, setIsSubmitting] = useState(false)

  const form = useForm<AnnouncementFormValues>({
    resolver: zodResolver(getAnnouncementFormSchema(t)),
    defaultValues: ANNOUNCEMENT_FORM_DEFAULT_VALUES,
  })

  useEffect(() => {
    if (!open) return
    if (announcement) {
      form.reset(transformAnnouncementToFormDefaults(announcement))
    } else {
      form.reset(ANNOUNCEMENT_FORM_DEFAULT_VALUES)
    }
  }, [open, announcement, form])

  const onSubmit = async (values: AnnouncementFormValues) => {
    setIsSubmitting(true)
    try {
      const payload = transformFormValuesToPayload(values)
      const result =
        isUpdate && announcement
          ? await updateAnnouncement({ ...payload, id: announcement.id })
          : await createAnnouncement(payload)
      if (result.success) {
        toast.success(
          isUpdate
            ? t('Announcement updated successfully')
            : t('Announcement created successfully')
        )
        onOpenChange(false)
        onSaved()
      } else {
        handleServerError(result)
      }
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={isUpdate ? t('Edit Announcement') : t('Create Announcement')}
      description={t(
        'Announcements are shown to users on the notice bar or as a popup.'
      )}
      contentClassName='sm:max-w-2xl'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button
            type='button'
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={isSubmitting}
          >
            {t('Cancel')}
          </Button>
          <Button
            form='announcement-form'
            type='submit'
            disabled={isSubmitting}
          >
            {isSubmitting ? t('Saving...') : t('Save changes')}
          </Button>
        </>
      }
    >
      <Form {...form}>
        <form
          id='announcement-form'
          onSubmit={form.handleSubmit(onSubmit)}
          className='space-y-4'
        >
          <FormField
            control={form.control}
            name='title'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Title')}</FormLabel>
                <FormControl>
                  <Input
                    {...field}
                    placeholder={t('Enter announcement title')}
                    maxLength={200}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='content'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Content')}</FormLabel>
                <FormControl>
                  <Textarea
                    {...field}
                    rows={5}
                    placeholder={t(
                      'Announcement content (supports Markdown & HTML)'
                    )}
                    maxLength={5000}
                  />
                </FormControl>
                <FormDescription>
                  {t('Supports Markdown and HTML')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className='grid gap-4 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name='type'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Type')}</FormLabel>
                  <Select
                    items={Object.entries(ANNOUNCEMENT_TYPE_LABELS).map(
                      ([value, label]) => ({ value, label: t(label) })
                    )}
                    value={field.value}
                    onValueChange={(value) =>
                      value !== null && field.onChange(value)
                    }
                  >
                    <FormControl>
                      <SelectTrigger className='w-full'>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent alignItemWithTrigger={false}>
                      {Object.entries(ANNOUNCEMENT_TYPE_LABELS).map(
                        ([value, label]) => (
                          <SelectItem key={value} value={value}>
                            {t(label)}
                          </SelectItem>
                        )
                      )}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='mode'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Announcement Method')}</FormLabel>
                  <Select
                    items={Object.entries(ANNOUNCEMENT_MODE_LABELS).map(
                      ([value, label]) => ({ value, label: t(label) })
                    )}
                    value={field.value}
                    onValueChange={(value) =>
                      value !== null && field.onChange(value)
                    }
                  >
                    <FormControl>
                      <SelectTrigger className='w-full'>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent alignItemWithTrigger={false}>
                      {Object.entries(ANNOUNCEMENT_MODE_LABELS).map(
                        ([value, label]) => (
                          <SelectItem key={value} value={value}>
                            {t(label)}
                          </SelectItem>
                        )
                      )}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <div className='grid gap-4 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name='publish_at'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Publish Time')}</FormLabel>
                  <FormControl>
                    <DateTimePicker
                      value={field.value}
                      onChange={field.onChange}
                      placeholder={t('Publish immediately')}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Leave empty to publish immediately')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='expire_at'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Expiration Time')}</FormLabel>
                  <FormControl>
                    <DateTimePicker
                      value={field.value}
                      onChange={field.onChange}
                      placeholder={t('Never expires')}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Leave empty for never expires')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <div className='grid gap-4 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name='daily_limit'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Daily Show Limit')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min='0'
                      value={field.value}
                      onChange={(e) =>
                        field.onChange(Number.parseInt(e.target.value, 10) || 0)
                      }
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Maximum popup shows per user per day (0 = unlimited)')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <div className='grid gap-4 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name='show_start_time'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Show Window Start')}</FormLabel>
                  <FormControl>
                    <Input type='time' {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='show_end_time'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Show Window End')}</FormLabel>
                  <FormControl>
                    <Input type='time' {...field} />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Daily show window in Beijing time (empty = all day)'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </form>
      </Form>
    </Dialog>
  )
}
