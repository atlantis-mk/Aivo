import { createFileRoute } from '@tanstack/react-router'
import { ProjectSelectionScreen } from '@/features/projects/project-selection-screen'

export const Route = createFileRoute('/projects')({
  component: ProjectSelectionScreen,
})
