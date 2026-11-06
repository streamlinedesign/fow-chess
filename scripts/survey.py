"""Draws the Survey piece set and tiles from the classic silhouettes.

Run from the repo root: python3 scripts/survey.py
Writes web/src/pieces/survey/{w,b}<kind>.svg and web/src/tiles/survey-{l,d}.svg.
Pieces are markers on a war-game map: White is parchment with sepia ink, Black
slate with brass lines, both with a hatched shadow down and to the right. Tiles
are flat map squares with a fine survey grid.
"""
import os
import re

SRC = OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'web', 'src')
KINDS = 'KQRBNPJX'
SIDES = {
    'w': dict(ol='#3b2a17', fill='#f1e6c6', ln='#3b2a17', sh='#3b2a17', ow=7.0),
    'b': dict(ol='#d4ad5a', fill='#262b26', ln='#d4ad5a', sh='#0b0d0b', ow=5.5),
}
def parse(kind):
    s = open(f'{SRC}/pieces/w{kind}.svg').read()
    groups = re.findall(r'<g [^>]*>(.*?)</g>', s)
    extra = s[s.rfind('</g>') + 4:s.rfind('</svg>')]
    return groups[1], groups[2], extra
def piece(side, kind):
    shapes, details, extra = parse(kind)
    t = SIDES[side]
    hatch = (f'<pattern id="h{side}{kind}" width="3" height="3" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">'
             f'<rect width="1.3" height="3" fill="{t["sh"]}"/></pattern>')
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="4 2 92 92"><defs>{hatch}</defs>'
            f'<g fill="url(#h{side}{kind})" stroke="url(#h{side}{kind})" stroke-width="{t["ow"]}" stroke-linejoin="round" opacity="0.7" transform="translate(3 2.5)">{shapes}</g>'
            f'<g fill="{t["ol"]}" stroke="{t["ol"]}" stroke-width="{t["ow"]}" stroke-linejoin="round">{shapes}</g>'
            f'<g fill="{t["fill"]}">{shapes}</g>'
            f'<g fill="none" stroke="{t["ln"]}" stroke-width="2.75" stroke-linecap="round">{details}</g>'
            f'{extra.replace("#050505", t["ln"]).replace("#fffabd", t["fill"])}</svg>')
def tile(bg, line, op, cross):
    """A flat map square: a fine survey grid, and on dark squares a crosshair mark at the centre."""
    grid = ''.join(f'<path d="M{v} 0 V100 M0 {v} H100"/>' for v in (25, 50, 75))
    mark = '<path d="M50 41 V59 M41 50 H59"/><circle cx="50" cy="50" r="4.5"/>' if cross else '<circle cx="50" cy="50" r="1.6" fill="LINE" stroke="none"/>'
    ticks = ''.join(f'<path d="M{x} {y} h{dx} M{x} {y} v{dy}"/>' for x, y, dx, dy in [(0, 0, 6, 6), (100, 0, -6, 6), (0, 100, 6, -6), (100, 100, -6, -6)])
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect width="100" height="100" fill="{bg}"/>'
            f'<g fill="none" stroke="{line}" stroke-width="0.6" opacity="{op * 0.5}">{grid}</g>'
            f'<g fill="none" stroke="{line}" stroke-width="1.4" opacity="{op}">{ticks}{mark.replace("LINE", line)}</g></svg>')
os.makedirs(f'{OUT}/pieces/survey', exist_ok=True); os.makedirs(f'{OUT}/tiles', exist_ok=True)
for side in 'wb':
    for k in KINDS:
        open(f'{OUT}/pieces/survey/{side}{k}.svg', 'w').write(piece(side, k))
open(f'{OUT}/tiles/survey-l.svg', 'w').write(tile('#e7dbb4', '#8a6a3a', 0.55, False))
open(f'{OUT}/tiles/survey-d.svg', 'w').write(tile('#7d8462', '#3e4430', 0.6, True))
