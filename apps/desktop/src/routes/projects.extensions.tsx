/* eslint-disable react-refresh/only-export-components */

import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/projects/extensions')({
  component: ProjectExtensionsRoute,
})

function ProjectExtensionsRoute() {
  return null
}
