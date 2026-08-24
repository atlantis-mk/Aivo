import { useEffect, useState } from 'react'
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
          void window.aivo?.toggleMaximize()
        }}
        aria-hidden="true"
      />
      <MacWindowControlPlaceholders />
      <Outlet />
    </AppConfigProvider>
  ),
})

function MacWindowControlPlaceholders() {
  const isMac = window.aivo?.platform === 'darwin'
  const [isFocused, setIsFocused] = useState(() => document.hasFocus())

  useEffect(() => {
    if (!isMac) return

    const handleFocus = () => setIsFocused(true)
    const handleBlur = () => setIsFocused(false)

    window.addEventListener('focus', handleFocus)
    window.addEventListener('blur', handleBlur)

    return () => {
      window.removeEventListener('focus', handleFocus)
      window.removeEventListener('blur', handleBlur)
    }
  }, [isMac])

  if (!isMac || isFocused) return null

  return (
    <div className="mac-window-control-placeholders" data-app-no-drag aria-hidden="true">
      <span className="mac-window-control-placeholder" />
      <span className="mac-window-control-placeholder" />
      <span className="mac-window-control-placeholder" />
    </div>
  )
}
