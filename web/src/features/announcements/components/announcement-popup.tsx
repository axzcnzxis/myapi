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
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'

import { getPublicPopupAnnouncement } from '../api'
import {
  canShowPopupToday,
  formatBeijingTime,
  incrementPopupShownCount,
} from '../lib'
import type { Announcement } from '../types'

/**
 * Global popup announcement. Mounted at the root so it renders both on the
 * pre-login home page and inside the authenticated console. The per-day show
 * limit is enforced locally (per id + Beijing calendar day).
 */
export function AnnouncementPopup() {
  const { t } = useTranslation()
  const [announcement, setAnnouncement] = useState<Announcement | null>(null)
  const [open, setOpen] = useState(false)
  const countedRef = useRef<number | null>(null)

  useEffect(() => {
    let cancelled = false
    void getPublicPopupAnnouncement()
      .then((res) => {
        if (cancelled) return
        const item = res?.data
        if (!item || !item.id) return
        if (!canShowPopupToday(item)) return
        setAnnouncement(item)
        setOpen(true)
      })
      .catch(() => {
        // The popup is best-effort; failures must never disturb the page.
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Count a show once per mounted announcement so refreshes still increment.
  useEffect(() => {
    if (open && announcement && countedRef.current !== announcement.id) {
      countedRef.current = announcement.id
      incrementPopupShownCount(announcement.id)
    }
  }, [open, announcement])

  return (
    <Dialog
      open={open}
      onOpenChange={setOpen}
      title={announcement?.title || t('Announcement')}
      description={
        announcement
          ? `${t('Published by')} ${announcement.publisher_name || '-'} · ${formatBeijingTime(
              announcement.published_at || announcement.publish_at
            )}`
          : undefined
      }
      contentClassName='sm:max-w-3xl'
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <Button type='button' onClick={() => setOpen(false)}>
          {t('Got it')}
        </Button>
      }
    >
      <ScrollArea className='max-h-[min(72vh,640px)] pr-4'>
        <RichContent breaks content={announcement?.content ?? ''} />
      </ScrollArea>
    </Dialog>
  )
}
