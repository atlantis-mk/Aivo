import { createRouter } from '@tanstack/react-router'
import { routeTree } from './routeTree.gen'
import { createDesktopRouterHistory } from './lib/router-history'

export const router = createRouter({
  routeTree,
  defaultPreload: 'intent',
  history: createDesktopRouterHistory(),
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
