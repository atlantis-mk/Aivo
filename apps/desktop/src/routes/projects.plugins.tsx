/* eslint-disable react-refresh/only-export-components */

import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/projects/plugins')({
  component: ProjectPluginsRoute,
})

function ProjectPluginsRoute() {
  return null
}
