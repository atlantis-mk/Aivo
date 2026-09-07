/* eslint-disable react-refresh/only-export-components */

import { useState } from 'react'
import { Link, createFileRoute } from '@tanstack/react-router'
import { AiCloud01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { ArrowLeft, RefreshCw, Settings } from 'lucide-react'
import { WindowControls } from '@/components/app-top-bar-controls'
import { Button } from '@/components/ui/button'
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
import { DesktopUpdateSettings } from '@/features/updates/desktop-update-settings'

export const Route = createFileRoute('/settings')({
  component: SettingsRoute,
})

function SettingsRoute() {
  const [section, setSection] = useState<'providers' | 'updates'>('providers')

  return (
    <div className="flex h-dvh min-h-0 flex-col overflow-hidden bg-background text-foreground">
      <header className="relative z-50 flex h-9 shrink-0 items-center gap-2 border-b px-3" data-app-drag>
        <WindowControls />
        <Button asChild size="sm" variant="ghost">
          <Link data-app-no-drag to="/projects/chat">
            <ArrowLeft data-icon="inline-start" />
            返回主页
          </Link>
        </Button>
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
                      aria-current={section === 'providers' ? 'page' : undefined}
                      className="gap-2.5 px-1.5 py-2 text-sm text-sidebar-foreground group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-2"
                      isActive={section === 'providers'}
                      onClick={() => setSection('providers')}
                    >
                      <HugeiconsIcon icon={AiCloud01Icon} strokeWidth={1.8} />
                      <span>模型提供商</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      aria-current={section === 'updates' ? 'page' : undefined}
                      className="gap-2.5 px-1.5 py-2 text-sm text-sidebar-foreground group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-2"
                      isActive={section === 'updates'}
                      onClick={() => setSection('updates')}
                    >
                      <RefreshCw />
                      <span>软件更新</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          </SidebarContent>
        </Sidebar>
        <SidebarInset className="h-full min-h-0 min-w-0 overflow-hidden">
          <main className="min-h-0 flex-1 bg-background">
            {section === 'providers' ? <ProviderSettingsScreen /> : <DesktopUpdateSettings />}
          </main>
        </SidebarInset>
      </SidebarProvider>
    </div>
  )
}
