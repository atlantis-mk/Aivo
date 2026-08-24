/* eslint-disable react-refresh/only-export-components */

import { Link, createFileRoute } from '@tanstack/react-router'
import { AiCloud01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { ArrowLeft, Settings } from 'lucide-react'
import { WindowControls } from '@/components/app-top-bar-controls'
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
} from '@/components/ui/sidebar'
import { ProviderSettingsScreen } from '@/features/providers/provider-settings-screen'

export const Route = createFileRoute('/settings')({
  component: SettingsRoute,
})

function SettingsRoute() {
  return (
    <div className="flex h-dvh min-h-0 flex-col overflow-hidden bg-background text-foreground">
      <header className="relative z-50 flex h-11 shrink-0 items-center gap-2 border-b px-3" data-app-drag>
        <WindowControls />
        <Link
          className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
          data-app-no-drag
          to="/projects/chat"
        >
          <ArrowLeft className="size-4" />
          返回主页
        </Link>
        <div className="pointer-events-none absolute left-1/2 flex -translate-x-1/2 items-center gap-2 text-sm font-semibold">
          <Settings className="size-4" />
          设置
        </div>
      </header>

      <SidebarProvider className="min-h-0 flex-1 overflow-hidden bg-background text-foreground">
        <Sidebar collapsible="none">
          <SidebarContent>
            <SidebarGroup>
              <SidebarGroupContent>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      aria-current="page"
                      className="gap-2.5 px-1.5 py-2 text-sm text-sidebar-foreground group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-2"
                      isActive
                    >
                      <HugeiconsIcon icon={AiCloud01Icon} strokeWidth={1.8} />
                      <span>模型提供商</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          </SidebarContent>
        </Sidebar>
        <SidebarInset className="h-full min-h-0 min-w-0 overflow-hidden">
          <main className="min-h-0 flex-1 bg-background">
            <ProviderSettingsScreen />
          </main>
        </SidebarInset>
      </SidebarProvider>
    </div>
  )
}
