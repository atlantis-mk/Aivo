import { createFileRoute } from '@tanstack/react-router'
import { SetupScreen } from '@/features/setup/setup-screen'

export const Route = createFileRoute('/setup')({
  component: SetupScreen,
})
