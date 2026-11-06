// The loading screen: check the game server, then hand the window over to it.
// The app sets window.__ETK_DESKTOP__ = { version, server } before this runs.
const app = window.__ETK_DESKTOP__ || { version: 'dev', server: 'http://localhost:4100/' }
const server = app.server.replace(/\/$/, '')
const $ = (id) => document.getElementById(id)

$('version').textContent = `version ${app.version}`
// The offline king reuses the logo's drawing.
document.querySelector('#offline .king').innerHTML = document.querySelector('.logo svg').innerHTML

const fill = (pct, step) => {
  $('fill').style.width = pct + '%'
  if (step) $('step').textContent = step
}

/** Asks the server who we are: up, and signed in or not. Throws when it can't be reached. */
async function reach() {
  const ctl = new AbortController()
  const timer = setTimeout(() => ctl.abort(), 6000)
  try {
    // no-cors: the answer is opaque from this page, but it arriving at all
    // means the server is up. The cookie is the game site's own, so signing
    // in happens there.
    await fetch(server + '/api/me', { mode: 'no-cors', cache: 'no-store', signal: ctl.signal })
  } finally {
    clearTimeout(timer)
  }
}

let wait = 2
let countdown = 0

async function start() {
  clearInterval(countdown)
  $('offline').hidden = true
  $('loading').hidden = false
  fill(15, 'Connecting')
  try {
    await reach()
  } catch {
    return offline()
  }
  fill(70, 'Signing you in')
  // Let the line catch up, then move over. The site paints on the same paper.
  setTimeout(() => {
    fill(100)
    location.replace(server + '/')
  }, 250)
}

/** Can't reach the server: say so, and try again on a growing timer. */
function offline() {
  $('loading').hidden = true
  $('offline').hidden = false
  wait = Math.min(wait * 2, 30)
  let left = wait
  const tell = () => ($('retry').textContent = `Check your connection. Trying again in ${left} s.`)
  tell()
  countdown = setInterval(() => {
    left -= 1
    if (left <= 0) start()
    else tell()
  }, 1000)
  $('now').focus()
}

$('now').addEventListener('click', start)
start()
