import { useEffect } from 'react'

import { hasAppBridge } from '@/lib/app-config'
import { scanGlobalSkills } from '@/services/aivo'

export function GlobalSkillScanner() {
  useEffect(() => {
    if (!hasAppBridge()) return
    void scanGlobalSkills().catch(() => undefined)
  }, [])

  return null
}
