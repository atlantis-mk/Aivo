import { createHashHistory, type RouterHistory } from '@tanstack/react-router'

export type DesktopRouterHistoryMode = 'browser' | 'hash'

export function desktopRouterHistoryMode(protocol: string): DesktopRouterHistoryMode {
  return protocol === 'file:' ? 'hash' : 'browser'
}

export function createDesktopRouterHistory(
  protocol = window.location.protocol,
): RouterHistory | undefined {
  return desktopRouterHistoryMode(protocol) === 'hash'
    ? createHashHistory()
    : undefined
}
