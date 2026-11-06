import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import { App } from './App'
import { DESKTOP } from './desktop'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)

// The service worker (public/sw.js) makes the site installable. Not in dev,
// where it would hold on to old files, and not in the desktop app, which is
// installed already.
if (import.meta.env.PROD && !DESKTOP && 'serviceWorker' in navigator) {
  addEventListener('load', () => navigator.serviceWorker.register('/sw.js').catch(() => {}))
}
