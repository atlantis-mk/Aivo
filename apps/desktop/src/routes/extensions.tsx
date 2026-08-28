/* eslint-disable react-refresh/only-export-components */

import { Link, createFileRoute } from '@tanstack/react-router'
import { ArrowLeft, Plug } from 'lucide-react'

import { WindowControls } from '@/components/app-top-bar-controls'
import { Button } from '@/components/ui/button'
import { ExtensionSettingsContent } from '@/features/projects/extension-settings-dialog'

export const Route = createFileRoute('/extensions')({
  component: ExtensionManagementRoute,
})

function ExtensionManagementRoute() {
  return (
    <div className="flex h-dvh min-h-0 flex-col overflow-hidden bg-background text-foreground">
      <header
        className="relative z-50 flex h-9 shrink-0 items-center gap-2 border-b px-3"
        data-app-drag
      >
        <WindowControls />
        <Button asChild size="sm" variant="ghost">
          <Link data-app-no-drag to="/projects/chat">
            <ArrowLeft data-icon="inline-start" />
            返回主页
          </Link>
        </Button>
        <div className="pointer-events-none absolute left-1/2 flex -translate-x-1/2 items-center gap-2 text-sm font-semibold">
          <Plug className="size-4" />
          扩展与 MCP
        </div>
      </header>

      <main className="min-h-0 flex-1 bg-background">
        <ExtensionSettingsContent className="min-h-0" surface="page" />
      </main>
    </div>
  )
}
