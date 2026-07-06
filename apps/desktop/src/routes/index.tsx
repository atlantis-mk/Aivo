/* eslint-disable react-refresh/only-export-components */

import { Navigate, createFileRoute } from '@tanstack/react-router'
import { SetupLoadingSkeleton } from '@/features/setup/setup-loading-skeleton'
import { useAppConfig } from '@/lib/app-config'

export const Route = createFileRoute('/')({
  component: IndexRedirect,
})

function IndexRedirect() {
  const { loading } = useAppConfig()

  if (loading) {
    return <SetupLoadingSkeleton />
  }

  return <Navigate to="/setup" replace />
}
