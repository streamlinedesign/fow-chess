"""Draws the cyberpunk piece set and tiles from the classic silhouettes.

Run from the repo root: python3 scripts/cyberpunk.py
Writes web/src/pieces/cyberpunk/{w,b}<kind>.svg and web/src/tiles/cyberpunk-{l,d}.svg.
White is ice white with dark lines, Black deep violet with neon cyan lines; both
have a magenta and cyan fringe.
"""
import os
import re

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'web', 'src')
KINDS = 'KQRBNPJX'

SIDES = {
    # outline, body, detail lines, fringe left and right, outline width
    'w': dict(ol='#0c0624', fill='#e6fbff', ln='#0c0624', fl='#ff2e97', fr='#00c8e0', ow=7.0),
    'b': dict(ol='#2ef2ff', fill='#150a2e', ln='#2ef2ff', fl='#ff3da5', fr='#2ef2ff', ow=5.0),
}


def parse(kind):
    """The classic white piece's shapes, detail lines and anything drawn after them."""
    s = open(f'{ROOT}/pieces/w{kind}.svg').read()
    groups = re.findall(r'<g [^>]*>(.*?)</g>', s)
    extra = s[s.rfind('</g>') + 4:s.rfind('</svg>')]
    return groups[1], groups[2], extra


def piece(side, kind):
    shapes, details, extra = parse(kind)
    t = SIDES[side]
    outline = lambda c, dx: (f'<g fill="{c}" stroke="{c}" stroke-width="{t["ow"]}" stroke-linejoin="round" opacity="0.9" '
                             f'transform="translate({dx} 0)">{shapes}</g>')
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="4 2 92 92">'
            f'{outline(t["fl"], -1.8)}{outline(t["fr"], 1.8)}'
            f'<g fill="{t["ol"]}" stroke="{t["ol"]}" stroke-width="{t["ow"]}" stroke-linejoin="round">{shapes}</g>'
            f'<g fill="{t["fill"]}">{shapes}</g>'
            f'<g fill="none" stroke="{t["ln"]}" stroke-width="2.75" stroke-linecap="round">{details}</g>'
            f'{extra.replace("#050505", t["ln"])}</svg>')


def tile(bg, bracket, op):
    """A flat square with neon brackets in its corners."""
    s = ''.join(f'<path d="M{x+rx*3} {y+ry*12} V{y+ry*3} H{x+rx*12}"/>'
                for x, y, rx, ry in [(0, 0, 1, 1), (100, 0, -1, 1), (0, 100, 1, -1), (100, 100, -1, -1)])
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect width="100" height="100" fill="{bg}"/>'
            f'<g fill="none" stroke="{bracket}" stroke-width="1.6" opacity="{op}">{s}</g></svg>')


os.makedirs(f'{ROOT}/pieces/cyberpunk', exist_ok=True)
for side in 'wb':
    for k in KINDS:
        open(f'{ROOT}/pieces/cyberpunk/{side}{k}.svg', 'w').write(piece(side, k))
open(f'{ROOT}/tiles/cyberpunk-l.svg', 'w').write(tile('#cdeef4', '#e0218a', 0.55))
open(f'{ROOT}/tiles/cyberpunk-d.svg', 'w').write(tile('#3a1d6e', '#ff3da5', 0.8))
