import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from '@tanstack/react-router'
import './index.css'
import { router } from './router'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Toaster } from '@/components/ui/sonner'
import { initializeSharedPreviewState } from '@/lib/preview-state'

async function bootstrap() {
  await initializeSharedPreviewState()
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <TooltipProvider>
        <RouterProvider router={router} />
        <Toaster />
      </TooltipProvider>
    </StrictMode>,
  )
}

void bootstrap()
