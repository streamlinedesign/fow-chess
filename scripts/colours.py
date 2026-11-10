"""Draws the colour sets and the black and white set from the classic pieces.

Run from the repo root: python3 scripts/colours.py
The colour sets keep the game's two tones and use colour only as the ink:
  pieces/light/  "<colour>-light", White's half: cream pieces drawn in the colour
  pieces/ink/    "<colour>-ink", Black's half: ink pieces with lines in the colour
  pieces/four/   the four-player seat sets, drawn like the light half
The black and white set ("mono") is plain white and plain black, with grey tiles:
  pieces/mono/{w,b}<kind>.svg and tiles/mono-{l,d}.svg
"""
import os
import re

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'web', 'src')
PIECES = os.path.join(ROOT, 'pieces')
KINDS = 'KQRBNPJX'
CREAM, INK = '#fffabd', '#050505'

# Lines on cream need a deep shade; lines on ink a lighter one.
ON_CREAM = {'r': '#b8402c', 'b': '#2c56a0', 'g': '#2e6e48', 'v': '#6e3a8e'}
ON_INK = {'r': '#e8705a', 'b': '#7a9ee6', 'g': '#6cba8a', 'v': '#b08ad8'}


def classic(side, kind):
    s = open(f'{PIECES}/{side}{kind}.svg').read()
    return re.sub(r'<metadata>.*?</metadata>', '', s, flags=re.S)


def write(path, svg):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    open(path, 'w').write(svg)


for k in KINDS:
    for c in 'rbgv':
        printed = classic('w', k).replace(INK, ON_CREAM[c])
        write(f'{PIECES}/light/{c}{k}.svg', printed)
        write(f'{PIECES}/four/{c}{k}.svg', printed)
        write(f'{PIECES}/ink/{c}{k}.svg', classic('b', k).replace(CREAM, ON_INK[c]))
    # Black and white: classic shapes in pure white and pure black.
    write(f'{PIECES}/mono/w{k}.svg', classic('w', k).replace(CREAM, '#ffffff').replace(INK, '#000000'))
    write(f'{PIECES}/mono/b{k}.svg', classic('b', k).replace(CREAM, '#ffffff').replace(INK, '#000000'))

tile = lambda fill: f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect width="100" height="100" fill="{fill}"/></svg>'
write(f'{ROOT}/tiles/mono-l.svg', tile('#eeeeee'))
write(f'{ROOT}/tiles/mono-d.svg', tile('#8c8c8c'))
