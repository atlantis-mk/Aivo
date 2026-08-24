import { createFileRoute } from '@tanstack/react-router'
import { ProjectWorkspaceScreen } from '@/features/projects/project-workspace-screen'

export const Route = createFileRoute('/projects')({
  component: ProjectWorkspaceScreen,
})
