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
import { Pencil, Plus, Send, Trash2, XCircle } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { NativeSelect } from '@/components/ui/native-select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { handleServerError } from '@/lib/handle-server-error'
import { cn } from '@/lib/utils'

import {
  cancelAnnouncement,
  deleteAnnouncement,
  getAnnouncements,
  publishAnnouncement,
} from './api'
import { AnnouncementFormDialog } from './components/announcement-form-dialog'
import {
  ANNOUNCEMENT_MODE_LABELS,
  formatBeijingTime,
  getAnnouncementStatusBadgeVariant,
  getAnnouncementStatusLabel,
} from './lib'
import { ANNOUNCEMENT_STATUSES } from './types'
import type { Announcement, AnnouncementScope } from './types'

const PAGE_SIZE = 10

const SCOPE_OPTIONS: { value: AnnouncementScope; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'current', label: 'Current' },
  { value: 'history', label: 'History' },
]

export function Announcements() {
  const { t } = useTranslation()
  const [scope, setScope] = useState<AnnouncementScope>('all')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<Announcement | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Announcement | null>(null)
  const [cancelTarget, setCancelTarget] = useState<Announcement | null>(null)
  const [busy, setBusy] = useState(false)

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['announcements', scope, status, page],
    queryFn: async () => {
      const res = await getAnnouncements({
        scope,
        status: status as never,
        p: page,
        page_size: PAGE_SIZE,
      })
      if (!res.success) {
        throw new Error(res.message || t('Failed to load announcements'))
      }
      return (
        res.data ?? { items: [], total: 0, page: 1, page_size: PAGE_SIZE }
      )
    },
  })

  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const openCreate = () => {
    setEditing(null)
    setFormOpen(true)
  }

  const openEdit = (announcement: Announcement) => {
    setEditing(announcement)
    setFormOpen(true)
  }

  const runAction = async (
    action: () => Promise<{ success: boolean; message?: string }>,
    successMessage: string,
    onDone?: () => void
  ) => {
    setBusy(true)
    try {
      const result = await action()
      if (result.success) {
        toast.success(t(successMessage))
        onDone?.()
        void refetch()
      } else {
        handleServerError(result)
      }
    } catch (error) {
      handleServerError(error)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Announcement Management')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <Button size='sm' onClick={openCreate}>
            <Plus className='h-4 w-4' />
            {t('New Announcement')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='flex flex-wrap items-center gap-2'>
            <div className='bg-muted inline-flex rounded-lg p-1'>
              {SCOPE_OPTIONS.map((option) => (
                <button
                  key={option.value}
                  type='button'
                  onClick={() => {
                    setScope(option.value)
                    setPage(1)
                  }}
                  className={cn(
                    'rounded-md px-3 py-1 text-sm transition-colors',
                    scope === option.value
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  {t(option.label)}
                </button>
              ))}
            </div>

            <NativeSelect
              value={status}
              onChange={(event) => {
                setStatus(event.target.value)
                setPage(1)
              }}
              className='w-40'
            >
              <option value=''>{t('All Status')}</option>
              {ANNOUNCEMENT_STATUSES.map((value) => (
                <option key={value} value={value}>
                  {t(getAnnouncementStatusLabel(value))}
                </option>
              ))}
            </NativeSelect>
          </div>

          <div className='mt-4 rounded-xl border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Title')}</TableHead>
                  <TableHead>{t('Announcement Method')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Publish Time')}</TableHead>
                  <TableHead>{t('Expiration Time')}</TableHead>
                  <TableHead>{t('Publisher')}</TableHead>
                  <TableHead className='text-right'>{t('Actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <TableRow>
                    <TableCell
                      colSpan={7}
                      className='text-muted-foreground h-24 text-center'
                    >
                      {t('Loading...')}
                    </TableCell>
                  </TableRow>
                ) : items.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={7}
                      className='text-muted-foreground h-24 text-center'
                    >
                      {t('No announcements found')}
                    </TableCell>
                  </TableRow>
                ) : (
                  items.map((item) => {
                    const canPublish =
                      item.status === 'draft' || item.status === 'scheduled'
                    const canCancel =
                      item.status === 'published' ||
                      item.status === 'scheduled'
                    return (
                      <TableRow key={item.id}>
                        <TableCell className='max-w-xs truncate font-medium'>
                          {item.title}
                        </TableCell>
                        <TableCell>
                          {t(ANNOUNCEMENT_MODE_LABELS[item.mode])}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={getAnnouncementStatusBadgeVariant(
                              item.status
                            )}
                          >
                            {t(getAnnouncementStatusLabel(item.status))}
                          </Badge>
                        </TableCell>
                        <TableCell data-table-text='secondary'>
                          {formatBeijingTime(
                            item.published_at || item.publish_at
                          )}
                        </TableCell>
                        <TableCell data-table-text='secondary'>
                          {item.expire_at
                            ? formatBeijingTime(item.expire_at)
                            : t('Never expires')}
                        </TableCell>
                        <TableCell data-table-text='secondary'>
                          {item.publisher_name || '-'}
                        </TableCell>
                        <TableCell className='text-right'>
                          <div className='flex justify-end gap-1'>
                            <Button
                              size='icon'
                              variant='ghost'
                              title={t('Edit')}
                              onClick={() => openEdit(item)}
                            >
                              <Pencil className='h-4 w-4' />
                            </Button>
                            {canPublish && (
                              <Button
                                size='icon'
                                variant='ghost'
                                title={t('Publish')}
                                disabled={busy}
                                onClick={() =>
                                  runAction(
                                    () => publishAnnouncement(item.id),
                                    'Announcement published successfully'
                                  )
                                }
                              >
                                <Send className='h-4 w-4' />
                              </Button>
                            )}
                            {canCancel && (
                              <Button
                                size='icon'
                                variant='ghost'
                                title={t('Cancel Announcement')}
                                onClick={() => setCancelTarget(item)}
                              >
                                <XCircle className='h-4 w-4' />
                              </Button>
                            )}
                            <Button
                              size='icon'
                              variant='ghost'
                              title={t('Delete')}
                              onClick={() => setDeleteTarget(item)}
                            >
                              <Trash2 className='text-destructive h-4 w-4' />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </div>

          {totalPages > 1 && (
            <div className='mt-4 flex items-center justify-end gap-2'>
              <Button
                size='sm'
                variant='outline'
                disabled={page <= 1}
                onClick={() => setPage((prev) => Math.max(1, prev - 1))}
              >
                {t('Previous')}
              </Button>
              <span className='text-muted-foreground text-sm'>
                {page} / {totalPages}
              </span>
              <Button
                size='sm'
                variant='outline'
                disabled={page >= totalPages}
                onClick={() =>
                  setPage((prev) => Math.min(totalPages, prev + 1))
                }
              >
                {t('Next')}
              </Button>
            </div>
          )}
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <AnnouncementFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        announcement={editing}
        onSaved={() => void refetch()}
      />

      <ConfirmDialog
        destructive
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        isLoading={busy}
        title={t('Delete Announcement')}
        desc={t('This action cannot be undone.')}
        confirmText={t('Delete')}
        handleConfirm={() => {
          if (!deleteTarget) return
          void runAction(
            () => deleteAnnouncement(deleteTarget.id),
            'Announcement deleted successfully',
            () => setDeleteTarget(null)
          )
        }}
      />

      <ConfirmDialog
        open={!!cancelTarget}
        onOpenChange={(open) => !open && setCancelTarget(null)}
        isLoading={busy}
        title={t('Cancel Announcement')}
        desc={t('Users will no longer see this announcement.')}
        confirmText={t('Cancel Announcement')}
        handleConfirm={() => {
          if (!cancelTarget) return
          void runAction(
            () => cancelAnnouncement(cancelTarget.id),
            'Announcement cancelled successfully',
            () => setCancelTarget(null)
          )
        }}
      />
    </>
  )
}
