"""Draws the Japanese piece set and tiles from the classic silhouettes.

Run from the repo root: python3 scripts/japanese.py
Writes web/src/pieces/japanese/{w,b}<kind>.svg and web/src/tiles/japanese-{l,d}.svg.
White is washi paper lined in indigo, Black is black lacquer lined in gold. Every piece
carries a red seal with its name: the shogi names where there is one (king, rook,
bishop, knight, pawn), a queen, a crown prince for the prince and "play" for the
jester pawn. Both squares carry the same wave pattern (seigaiha), on tatami and on indigo.
"""
import os
import re

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'web', 'src')
KINDS = 'KQRBNPJX'

# Each name as an outline in a 10 by 10 box, so no font is needed to draw or to show it.
# The outlines are IPAGothic's (IPA Font License 1.0).
GLYPHS = {
    'K': ('王', 'M5.35 2.21V4.62H8.35V5.29H5.35V8.05H9.2V8.74H0.8V8.05H4.57V5.29H1.65V4.62H4.57V2.21H1.15V1.53H8.84V2.21Z'),
    'Q': ('后', 'M1.67 1.51 2.01 1.5Q5.35 1.41 8 0.91L8.6 1.58Q5.85 2.05 2.55 2.15H2.41V3.39H9.4V4.07H2.41V4.36Q2.41 6.28 2.16 7.4Q1.9 8.56 1.25 9.48L0.64 8.92Q1.67 7.59 1.67 4.79ZM8.43 5.37V9.5H7.73V8.8H3.81V9.5H3.11V5.37ZM3.81 6.02V8.15H7.73V6.02Z'),
    'R': ('飛', 'M6.87 2.09Q6.95 2.11 7.36 2.28Q7.73 1.79 8.05 1.21L8.66 1.54Q8.13 2.24 7.86 2.51Q8.37 2.76 8.83 3.05L8.45 3.57Q7.86 3.13 7.02 2.7Q7.22 3.33 7.57 3.75Q8.04 4.32 8.36 4.32Q8.54 4.32 8.77 3.5L8.81 3.38L9.37 3.75Q9.02 5.06 8.51 5.06Q7.89 5.06 7.22 4.36Q6.34 3.45 6.12 1.63H1.25V1.02H6.75Q6.76 1.6 6.87 2.09ZM3.16 4.87H4.45V1.96H5.14V4.87H7.04Q7.04 5.51 7.1 6.07L7.38 6.19Q7.75 5.74 8.06 5.26L8.65 5.61Q8.33 6.01 7.9 6.42Q8.38 6.64 9.08 7.09L8.71 7.64Q7.89 7.02 7.18 6.65Q7.35 7.49 7.8 8.06Q8.23 8.61 8.48 8.61Q8.75 8.61 8.96 7.45L9.56 7.85Q9.22 9.41 8.62 9.41Q8.34 9.41 7.81 9.03Q6.48 8.1 6.38 5.47H5.14V9.5H4.45V5.47H3.15Q3.09 6.79 2.78 7.63Q2.34 8.8 1.32 9.6L0.83 9.09Q2.37 7.93 2.47 5.51H0.74V4.92H2.48V3.3Q1.61 3.55 1.08 3.65L0.76 3.08Q2.51 2.77 3.8 2.08L4.25 2.61Q3.8 2.84 3.24 3.05Q3.18 3.07 3.16 3.08Z'),
    'B': ('角', 'M5.79 2.86H8.38V8.57Q8.38 9.04 8.19 9.22Q8.01 9.38 7.55 9.38Q6.66 9.38 5.87 9.31L5.73 8.54Q6.65 8.67 7.35 8.67Q7.68 8.67 7.68 8.3V6.85H2.6Q2.46 8.59 1.32 9.59L0.8 9.06Q1.62 8.38 1.81 7.41Q1.91 6.91 1.91 6.02V3.58Q1.64 3.83 1.14 4.24L0.65 3.71Q2.59 2.32 3.33 0.49L4.11 0.64Q3.96 0.96 3.84 1.17H6.43L6.85 1.49Q6.31 2.25 5.79 2.86ZM5.43 3.46V4.5H7.68V3.46ZM5.43 5.08V6.25H7.68V5.08ZM4.75 6.25V5.08H2.62V6.25ZM4.75 4.5V3.46H2.62V4.5ZM4.97 2.86Q5.38 2.42 5.81 1.77H3.47Q3.09 2.32 2.62 2.86Z'),
    'N': ('桂', 'M2 4.68Q1.57 6.31 0.78 7.57L0.36 6.85Q1.4 5.35 1.91 3.26H0.57V2.63H2V0.51H2.68V2.63H3.95V3.26H2.68V4.24Q3.43 4.91 4.07 5.66L3.67 6.37Q3.23 5.67 2.68 5.01V9.5H2ZM6.89 4.11H9.31V4.75H3.92V4.11H6.2V2.63H4.32V2.02H6.2V0.61H6.89V2.02H8.91V2.63H6.89ZM6.2 6.28V5.06H6.89V6.28H8.91V6.91H6.89V8.49H9.44V9.14H3.82V8.49H6.2V6.91H4.33V6.28Z'),
    'P': ('歩', 'M5.35 1.94H8.24V2.56H5.35V3.78H9.29V4.42H5.35V6.73Q5.35 7.12 5.17 7.25Q4.98 7.39 4.5 7.39Q4 7.39 3.55 7.34L3.45 6.65Q4 6.74 4.35 6.74Q4.63 6.74 4.63 6.39V4.42H0.7V3.78H2.24V1.57H2.94V3.78H4.63V0.5H5.35ZM0.87 7.01Q2 6.07 2.73 4.8L3.39 5.09Q2.58 6.5 1.46 7.5ZM8.71 7.27Q7.84 6.09 6.65 5.17L7.24 4.75Q8.47 5.68 9.34 6.71ZM1.42 8.92Q5.32 8.47 7.02 6.25L7.65 6.69Q5.57 9.05 1.88 9.59Z'),
    'J': ('戯', 'M7.6 6.35Q8.18 5.33 8.48 4.33L9.08 4.61Q8.57 6.08 7.83 7.18Q8.27 8.47 8.57 8.47Q8.79 8.47 8.88 7.08L8.9 6.86L9.53 7.29Q9.34 9.37 8.71 9.37Q8 9.37 7.36 7.83Q6.52 8.88 5.68 9.57L5.19 9.03Q6.32 8.24 7.11 7.11Q6.76 5.88 6.64 3.99L5.6 4.07L5.54 3.42L6.6 3.35Q6.53 2.3 6.53 0.61H7.2Q7.2 2.02 7.25 3.22V3.3L9.5 3.15L9.52 3.76L7.28 3.94Q7.39 5.52 7.6 6.35ZM3.34 2.89V3.6H4.82V4.13H3.34V4.46Q3.34 4.77 4.03 4.77Q4.7 4.77 4.85 4.67Q4.97 4.6 5.03 4.13Q5.03 4.04 5.04 4.01L5.62 4.2Q5.58 4.99 5.3 5.15Q5.07 5.31 3.98 5.31Q3.18 5.31 2.93 5.17Q2.72 5.04 2.72 4.71V4.24H1.79V3.71H2.72V2.89H1.57V4.43Q1.57 6.71 1.44 7.64Q1.31 8.7 0.85 9.6L0.34 9.1Q0.75 8.31 0.85 7.14Q0.93 6.23 0.93 4.37V2.32H2.79V0.5H3.46V1.2H5.35V1.76H3.46V2.32H5.38L5.78 2.59Q5.56 3.26 5.35 3.64L4.81 3.41Q4.97 3.14 5.05 2.89ZM3.45 8.28 3.88 8.18V5.64H4.48V8.04Q5.37 7.81 5.89 7.67L5.92 8.21Q4.06 8.83 1.75 9.28L1.51 8.67Q1.59 8.66 2.39 8.5Q2.72 8.44 2.85 8.41V5.64H3.45ZM2.07 8.13Q1.96 7.02 1.77 6.2L2.36 6.05Q2.56 6.83 2.67 7.92ZM4.58 7.53Q4.92 6.6 5.07 5.85L5.69 6Q5.56 6.7 5.13 7.67ZM8.51 2.76Q8.12 1.77 7.69 1.12L8.25 0.85Q8.74 1.62 9.09 2.46Z'),
    'X': ('太', 'M5.43 3.73Q5.88 5.22 6.98 6.51Q7.98 7.7 9.38 8.4L8.81 9.11Q6.01 7.29 5.03 4.41Q4.84 5.55 4.21 6.6L4.16 6.69Q5.17 7.37 6.05 8.22L5.49 8.87Q4.73 8.01 3.75 7.26Q2.7 8.58 1.36 9.35L0.81 8.69Q4.06 7.1 4.49 3.73H0.75V3.04H4.52V0.67H5.32V3.04H9.25V3.73Z'),
}

# Where the seal's centre sits on each piece, and how wide it is.
SEAL = {'K': (50, 64, 16), 'Q': (50, 64, 16), 'R': (50, 62, 16), 'B': (50, 64, 16),
        'N': (53, 66, 16), 'P': (50, 70, 13), 'J': (50, 70, 13), 'X': (50, 64, 16)}

SIDES = {
    # body, outline and lines, outline width, the seal's character, sheen on the left, shade on the right
    'w': dict(body='#efe4cb', line='#1d2a4a', ow=6.5, ink='#efe4cb', sheen=0.25, shade=0.18),
    'b': dict(body='#161213', line='#c9a14a', ow=4.5, ink='#f6e7c0', sheen=0.08, shade=0.3),
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
    i = f'jp{side}{kind}'
    x, y, w = SEAL[kind]
    g = (w - 3.2) / 10
    seal = (f'<rect x="{x - w / 2}" y="{y - w / 2}" width="{w}" height="{w}" rx="1.4" fill="#c4372c"/>'
            f'<path transform="translate({x - w / 2 + 1.6} {y - w / 2 + 0.8}) scale({g:.2f})" d="{GLYPHS[kind][1]}" fill="{t["ink"]}"/>')
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="4 2 92 92"><defs><clipPath id="{i}c">{shapes}</clipPath>'
            f'<linearGradient id="{i}g" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="#fff" stop-opacity="{t["sheen"]}"/>'
            f'<stop offset="0.55" stop-color="#fff" stop-opacity="0"/><stop offset="1" stop-color="#000" stop-opacity="{t["shade"]}"/></linearGradient></defs>'
            f'<g fill="{t["line"]}" stroke="{t["line"]}" stroke-width="{t["ow"]}" stroke-linejoin="round">{shapes}</g>'
            f'<g fill="{t["body"]}">{shapes}</g>'
            f'<g clip-path="url(#{i}c)"><rect width="100" height="100" fill="url(#{i}g)"/></g>'
            f'<g fill="none" stroke="{t["line"]}" stroke-width="2.75" stroke-linecap="round">{details}</g>'
            f'{extra.replace("#050505", t["line"])}{seal}</svg>')


def tile(bg, wave, op):
    """Seigaiha: rows of overlapping wave crests."""
    waves = ''
    for row in range(-1, 6):
        for col in range(-1, 6):
            cx, cy = col * 20 + (10 if row % 2 else 0), row * 10 + 10
            waves += ''.join(f'<path d="M{cx - r} {cy} A{r} {r} 0 0 1 {cx + r} {cy}"/>' for r in (10, 7, 4))
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect width="100" height="100" fill="{bg}"/>'
            f'<g fill="none" stroke="{wave}" stroke-width="0.8" opacity="{op}">{waves}</g></svg>')


os.makedirs(f'{ROOT}/pieces/japanese', exist_ok=True)
for side in 'wb':
    for k in KINDS:
        open(f'{ROOT}/pieces/japanese/{side}{k}.svg', 'w').write(piece(side, k))
open(f'{ROOT}/tiles/japanese-l.svg', 'w').write(tile('#d9c99a', '#8a7440', 0.5))
open(f'{ROOT}/tiles/japanese-d.svg', 'w').write(tile('#22355f', '#9fb3d6', 0.35))
