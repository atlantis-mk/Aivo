import { Outlet, createRootRoute } from '@tanstack/react-router'
import { GlobalSkillScanner } from '@/components/global-skill-scanner'
import { AppConfigProvider } from '@/lib/app-config'

export const Route = createRootRoute({
  component: () => (
    <AppConfigProvider>
      <GlobalSkillScanner />
      <div
        className="window-title-drag-region"
        data-app-drag
        onDoubleClick={() => {
          void window.aivoDesktop?.window.toggleMaximize()
        }}
        aria-hidden="true"
      />
      <Outlet />
    </AppConfigProvider>
  ),
})
