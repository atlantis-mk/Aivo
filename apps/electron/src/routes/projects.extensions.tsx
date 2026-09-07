/* eslint-disable react-refresh/only-export-components */

import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/projects/extensions')({
  beforeLoad: () => {
    throw redirect({ to: '/extensions' })
  },
})
