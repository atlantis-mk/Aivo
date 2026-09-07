/* eslint-disable react-refresh/only-export-components */

import { useEffect } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { SetupLoadingSkeleton } from '@/features/setup/setup-loading-skeleton'
import { startupRouteFor } from '@/features/setup/setup-routing'
import { useAppConfig } from '@/lib/app-config'

export const Route = createFileRoute('/')({
  component: IndexRedirect,
})

function IndexRedirect() {
  const { config, loading } = useAppConfig()
  const navigate = useNavigate()

  useEffect(() => {
    if (loading) return
    void navigate({ to: startupRouteFor(config), replace: true })
  }, [config, loading, navigate])

  return <SetupLoadingSkeleton />
}
