function randomId() {
  // crypto.randomUUID needs HTTPS, which a phone on the LAN won't have.
  const b = crypto.getRandomValues(new Uint8Array(16))
  return Array.from(b, (x) => x.toString(16).padStart(2, '0')).join('')
}

// A stable per-browser ID so a refresh or reconnect gets the same seat back.
export const playerId: string = (() => {
  try {
    let id = localStorage.getItem('playerId')
    if (!id) {
      id = randomId()
      localStorage.setItem('playerId', id)
    }
    return id
  } catch {
    return randomId()
  }
})()
