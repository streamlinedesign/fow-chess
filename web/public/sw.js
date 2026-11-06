// The service worker that makes the site installable and quick to open again.
// It only ever serves the app's own files: the API, live games (/ws), and link
// preview pictures (/og) always go to the network, so a game in progress sees
// exactly what it would without it. Pages come from the network first and
// fall back to the last copy when offline; built files under /assets have
// their hash in the name, so a copy is good for ever.
const CACHE = 'etk-1'
const SHELL = '/'
const MAX_ASSETS = 300

self.addEventListener('install', (e) => {
  e.waitUntil(caches.open(CACHE).then((c) => c.add(SHELL)).then(() => self.skipWaiting()))
})

self.addEventListener('activate', (e) => {
  e.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  )
})

self.addEventListener('fetch', (e) => {
  const req = e.request
  const url = new URL(req.url)
  if (req.method !== 'GET' || url.origin !== location.origin) return
  if (/^\/(api|ws|og)\//.test(url.pathname)) return

  if (req.mode === 'navigate') {
    e.respondWith(
      fetch(req)
        .then((res) => {
          if (res.ok) {
            const copy = res.clone()
            caches.open(CACHE).then((c) => c.put(SHELL, copy))
          }
          return res
        })
        .catch(() => caches.match(SHELL).then((r) => r || Response.error())),
    )
    return
  }

  if (url.pathname.startsWith('/assets/')) {
    e.respondWith(
      caches.match(req).then(
        (hit) =>
          hit ||
          fetch(req).then((res) => {
            if (res.ok) {
              const copy = res.clone()
              caches.open(CACHE).then((c) => c.put(req, copy).then(() => trim(c)))
            }
            return res
          }),
      ),
    )
    return
  }

  // Icons, the manifest and other files at the root: the copy at once, a fresh one for next time.
  e.respondWith(
    caches.match(req).then((hit) => {
      const fresh = fetch(req)
        .then((res) => {
          if (res.ok) {
            const copy = res.clone()
            caches.open(CACHE).then((c) => c.put(req, copy))
          }
          return res
        })
        .catch(() => hit || Response.error())
      return hit || fresh
    }),
  )
})

// Old builds' files pile up over releases; drop the oldest past MAX_ASSETS.
async function trim(cache) {
  const keys = (await cache.keys()).filter((r) => new URL(r.url).pathname.startsWith('/assets/'))
  await Promise.all(keys.slice(0, Math.max(0, keys.length - MAX_ASSETS)).map((k) => cache.delete(k)))
}
