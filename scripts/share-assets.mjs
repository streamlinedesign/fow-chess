// Draws the pictures the share package needs from the SVGs in the repo:
// the app icons in web/public from the favicon (the header's king, turned),
// and the pieces and king the server draws link previews with, in server/og.
// Run with `make share-assets` after changing the favicon or the pieces;
// the PNGs are committed so builds don't need a browser.
import { readFileSync, writeFileSync } from 'node:fs'
import { createRequire } from 'node:module'

// Playwright is not a dependency of the app: install it once with `npm i -g playwright`.
const { chromium } = createRequire(import.meta.url)('playwright')

const root = new URL('..', import.meta.url).pathname
const read = (p) => readFileSync(root + p, 'utf8')

// The favicon in its light colours, whatever the system theme.
const king = read('web/public/favicon.svg').replace(/@media[^{]*\{(?:[^{}]*\{[^}]*\})*[^}]*\}/, '')
const dataURL = (svg) => 'data:image/svg+xml;base64,' + Buffer.from(svg).toString('base64')

const browser = await chromium.launch()
const page = await browser.newPage()

async function shoot(out, size, html) {
  await page.setViewportSize({ width: size, height: size })
  await page.setContent(`<style>html,body{margin:0;width:${size}px;height:${size}px;overflow:hidden}</style>${html}`)
  await page.waitForFunction(() => [...document.images].every((i) => i.complete))
  await page.screenshot({ path: root + out, omitBackground: true })
  console.log(out)
}

// App icons: the king on paper. Maskable icons keep it inside the middle 80%.
const icon = (size, pad, round) =>
  `<div style="width:${size}px;height:${size}px;background:#fffabd;display:grid;place-items:center;border-radius:${round}">` +
  `<img src="${dataURL(king)}" style="width:${size * (1 - 2 * pad)}px;height:${size * (1 - 2 * pad)}px"></div>`
await shoot('web/public/icon-192.png', 192, icon(192, 0.08, '0'))
await shoot('web/public/icon-512.png', 512, icon(512, 0.08, '0'))
await shoot('web/public/icon-maskable-512.png', 512, icon(512, 0.2, '0'))
await shoot('web/public/apple-touch-icon.png', 180, icon(180, 0.12, '0'))

// Link preview parts: each piece of the default set, and the turned king.
for (const c of 'wb')
  for (const k of 'KQRBNPJX') {
    const svg = read(`web/src/pieces/${c}${k}.svg`)
    await shoot(`server/og/pieces/${c}${k}.png`, 128, `<img src="${dataURL(svg)}" style="width:128px;height:128px">`)
  }
await shoot('server/og/king.png', 128, `<img src="${dataURL(king)}" style="width:128px;height:128px">`)

await browser.close()
