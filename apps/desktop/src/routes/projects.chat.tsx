/* eslint-disable react-refresh/only-export-components */

import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/projects/chat')({
  component: ProjectChatRoute,
})

function ProjectChatRoute() {
  return null
}
