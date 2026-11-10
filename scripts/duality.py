"""Draws the grandmaster set, Duality, and its tiles from the classic pieces.

Run from the repo root: python3 scripts/duality.py
Writes web/src/pieces/grandmaster/{w,b}<kind>.svg and web/src/tiles/grandmaster-{l,d}.svg.
Cream and ink, each holding part of the other, like yin and yang: a White piece is
cream with its outer side in ink, along an S curve; a Black piece is the mirror. So
the side is never in doubt, the larger part of the piece and its whole base stay in the
side's own tone, and the outline is the classic one: ink round White, cream round Black.
"""
import os
import re

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'web', 'src')
KINDS = 'KQRBNPJX'
CREAM, INK = '#fffabd', '#050505'


def parse(kind):
    """The classic white piece's shapes, detail lines and anything drawn after them."""
    s = open(f'{ROOT}/pieces/w{kind}.svg').read()
    groups = re.findall(r'<g [^>]*>(.*?)</g>', s)
    extra = s[s.rfind('</g>') + 4:s.rfind('</svg>')]
    return groups[1], groups[2], extra


def piece(side, kind):
    shapes, details, extra = parse(kind)
    own, other = (CREAM, INK) if side == 'w' else (INK, CREAM)
    i = f'du{side}{kind}'
    # The other tone takes the outer third of the piece, along an S from top to bottom,
    # and stops at the base, which stays whole in the side's own tone.
    x = 55 if side == 'w' else 45
    edge = 100 if side == 'w' else 0
    curl = f'M{x} 0 C{x + 8} 16 {x - 8} 32 {x} 46 C{x + 8} 60 {x - 8} 70 {x} 78 H{edge} V0 Z'
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="4 2 92 92"><defs>'
            f'<clipPath id="{i}c">{shapes}</clipPath><clipPath id="{i}o"><path d="{curl}"/></clipPath></defs>'
            f'<g fill="{other}" stroke="{other}" stroke-width="{7.0 if side == "w" else 5.0}" stroke-linejoin="round">{shapes}</g>'
            f'<g fill="{own}">{shapes}</g>'
            f'<g clip-path="url(#{i}c)"><g clip-path="url(#{i}o)"><rect width="100" height="100" fill="{other}"/></g></g>'
            f'<g fill="none" stroke="{other}" stroke-width="2.75" stroke-linecap="round">{details}</g>'
            f'<g clip-path="url(#{i}o)"><g fill="none" stroke="{own}" stroke-width="2.75" stroke-linecap="round">{details}</g></g>'
            f'{extra.replace(INK, other)}</svg>')


def tile(light):
    """A square in its own tone, the other tone curling in from a corner, a seed of each in the other."""
    a, b = ('#fbf5bf', '#1a1a16') if light else ('#2a2a24', '#fbf5bf')
    op = 0.14 if light else 0.12
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect width="100" height="100" fill="{a}"/>'
            f'<path d="M100 40 C70 40 70 70 40 70 C25 70 15 85 15 100 H100 Z" fill="{b}" opacity="{op}"/>'
            f'<circle cx="74" cy="76" r="4" fill="{a}" opacity="0.9"/><circle cx="30" cy="34" r="4" fill="{b}" opacity="{op + 0.02}"/></svg>')


for side in 'wb':
    for k in KINDS:
        open(f'{ROOT}/pieces/grandmaster/{side}{k}.svg', 'w').write(piece(side, k))
open(f'{ROOT}/tiles/grandmaster-l.svg', 'w').write(tile(True))
open(f'{ROOT}/tiles/grandmaster-d.svg', 'w').write(tile(False))
