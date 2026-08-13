/* eslint-disable react-refresh/only-export-components */

import { Link, createFileRoute } from '@tanstack/react-router'
import { AiCloud01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { ArrowLeft } from 'lucide-react'
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
} from '@/components/ui/sidebar'
import { cn } from '@/lib/utils'
import { ProviderSettingsScreen } from '@/features/providers/provider-settings-screen'

export const Route = createFileRoute('/settings')({
  component: SettingsRoute,
})

function SettingsRoute() {
  const isMac = window.aivo?.platform === 'darwin'

  return (
    <SidebarProvider className="h-dvh !min-h-0 overflow-hidden bg-background text-foreground">
      <Sidebar collapsible="none">
        <SidebarHeader className="h-9 shrink-0 p-0" data-app-drag>
          <SettingsWindowControls isMac={isMac} />
        </SidebarHeader>
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    asChild
                    className="gap-2.5 px-1.5 py-2 text-sm text-sidebar-foreground group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-2"
                  >
                    <Link to="/projects/chat">
                      <ArrowLeft />
                      <span>返回主页</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
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
        <div className="h-9 shrink-0 bg-background" data-app-drag />
        <main className="min-h-0 flex-1 bg-background">
          <ProviderSettingsScreen />
        </main>
      </SidebarInset>
    </SidebarProvider>
  )
}

function SettingsWindowControls({ isMac }: { isMac: boolean }) {
  return (
    <div
      aria-hidden="true"
      className={cn("h-full shrink-0", isMac ? "w-[54px]" : "w-0")}
      data-app-no-drag
    />
  )
}
