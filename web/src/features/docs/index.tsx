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
import { RefreshIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Markdown } from '@/components/ui/markdown'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { useStatus } from '@/hooks/use-status'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { fetchDocsMarkdown } from './api'
import { resolveDocsSource } from './source'

const DOCS_STALE_TIME = 5 * 60 * 1000

/**
 * In-app documentation page.
 *
 * The dashboard renders the repository markdown itself instead of sending the
 * visitor to GitHub. Only the initial root administrator may force a fresh pull,
 * which revalidates the document against its upstream revision.
 */
export function Docs() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const isRoot = useAuthStore(
    (state) => (state.auth.user?.role ?? 0) >= ROLE.SUPER_ADMIN
  )
  const docsLink = status?.docs_link as string | undefined
  const source = useMemo(() => resolveDocsSource(docsLink), [docsLink])
  const query = useQuery({
    queryKey: ['docs-markdown', source.rawUrl],
    queryFn: ({ signal }) => fetchDocsMarkdown(source, signal),
    retry: false,
    staleTime: DOCS_STALE_TIME,
  })

  let body: React.ReactNode
  if (query.isPending) {
    body = (
      <div className='flex flex-col gap-4'>
        <Skeleton className='h-8 w-[45%]' />
        <Skeleton className='h-4 w-full' />
        <Skeleton className='h-4 w-[90%]' />
        <Skeleton className='h-4 w-[80%]' />
      </div>
    )
  } else if (query.isError) {
    body = (
      <div className='space-y-3'>
        <p className='text-muted-foreground' role='alert'>
          {t('Failed to load')}
        </p>
        <Button
          type='button'
          variant='outline'
          disabled={query.isFetching}
          onClick={() => void query.refetch()}
        >
          {t('Retry')}
        </Button>
      </div>
    )
  } else {
    body = (
      <Markdown baseUrl={source.linkBaseUrl} className='max-w-none'>
        {query.data}
      </Markdown>
    )
  }

  return (
    <PublicLayout>
      <div className='mx-auto max-w-4xl px-4 py-8'>
        <div className='mb-6 flex flex-wrap items-start justify-between gap-3'>
          <div className='min-w-0'>
            <h1 className='text-2xl font-semibold'>{t('Docs')}</h1>
            <p className='text-muted-foreground truncate text-sm'>
              {source.repository}
            </p>
          </div>
          {isRoot && (
            <Button
              type='button'
              variant='outline'
              aria-busy={query.isFetching}
              disabled={query.isFetching}
              onClick={() => void query.refetch()}
            >
              {query.isFetching ? (
                <Spinner data-icon='inline-start' aria-hidden='true' />
              ) : (
                <HugeiconsIcon
                  icon={RefreshIcon}
                  data-icon='inline-start'
                  aria-hidden='true'
                />
              )}
              {t('Pull latest')}
            </Button>
          )}
        </div>
        {body}
      </div>
    </PublicLayout>
  )
}
