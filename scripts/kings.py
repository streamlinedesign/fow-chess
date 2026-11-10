"""Draws the four proposed kings in every piece set, from each set's own king or pawn.

Run from the repo root: python3 scripts/kings.py
Writes design/kings/<set>/<side><kind>.svg and design/kings/index.html (a contact sheet).
Every set draws its king (and pawn) from the classic silhouettes, so a new king is
that set's file with the classic shapes and detail lines swapped for the new ones.

  E  the Eye: a plain eye with a slit pupil and a small cross, on the king's body
  S  the swap king: a rook's battlements for a crown
  H  the pawn king: a pawn with a small cross on its head
  I  the infinity king: an infinity sign where the cross was

The files stay out of web/src until the rules for them ship, so the app doesn't carry
art it can't use. See design/kings/README.md.
"""
import os
import re

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..')
PIECES = os.path.join(ROOT, 'web', 'src', 'pieces')
OUT = os.path.join(ROOT, 'design', 'kings')

# The classic king, split into what a new king keeps and what it replaces.
BASE = '<rect x="22" y="78" width="56" height="10" rx="2.5"/>'
BODY = '<path d="M32 78 C37 66 41 58 42 50 H58 C59 58 63 66 68 78 Z"/>'
COLLAR = '<rect x="35" y="44" width="30" height="7" rx="2"/>'
CUP = '<path d="M33 45 L29 29 C36 24 64 24 71 29 L67 45 Z"/>'
CROSS = '<path d="M46.75 27 V20 H40.5 V14 H46.75 V8 H53.25 V14 H59.5 V20 H53.25 V27 Z"/>'
K_SHAPES = BASE + BODY + COLLAR + CUP + CROSS
K_LINES = ('<path d="M22 78 H78"/><path d="M35 51 H65"/><path d="M33 45 H67"/>'
           '<path d="M34 32.5 C41 29.5 59 29.5 66 32.5"/>')

P_SHAPES = ('<rect x="27" y="78" width="46" height="10" rx="2.5"/>'
            '<path d="M35 78 C39 69 42 63 43 58 H57 C58 63 61 69 65 78 Z"/>'
            '<rect x="37" y="52" width="26" height="7" rx="2"/><circle cx="50" cy="38" r="12.5"/>')
P_LINES = '<path d="M27 78 H73"/><path d="M37 59 H63"/>'

# The Eye: a plain almond eye on the collar where the cup was, a small cross on top.
ALMOND = '<path d="M24 34 C31 17 69 17 76 34 C69 51 31 51 24 34 Z"/>'
EYE_CROSS = '<path d="M47.75 23 V17 H43.5 V12.5 H47.75 V8 H52.25 V12.5 H56.5 V17 H52.25 V23 Z"/>'
EYE_SHAPES = BASE + BODY + COLLAR + EYE_CROSS + ALMOND
EYE_LINES = ('<path d="M22 78 H78"/><path d="M35 51 H65"/>'
             '<circle cx="50" cy="34" r="8.5"/>'
             '<path d="M50 28 V40" stroke-width="3"/>')

# The swap king: battlements for a crown.
BATTLEMENTS = '<path d="M36 28 V13 H42.5 V18 H47 V13 H53 V18 H57.5 V13 H64 V28 Z"/>'
SWAP_SHAPES = BASE + BODY + COLLAR + CUP + BATTLEMENTS
SWAP_LINES = K_LINES

# The pawn king: the pawn as it is, a small cross on its head.
SMALL_CROSS = '<path d="M47.5 27 V22 H43 V17.5 H47.5 V12.5 H52.5 V17.5 H57 V22 H52.5 V27 Z"/>'
PAWN_SHAPES = P_SHAPES + SMALL_CROSS
PAWN_LINES = P_LINES

# The infinity king: a thick infinity sign on a short stem over the cup.
INFINITY = ('<path fill-rule="evenodd" d="M50 13.6 C46.5 10.4 44 9.6 41.5 9.6 C36.6 9.6 33.2 12.8 33.2 17 '
            'C33.2 21.2 36.6 24.4 41.5 24.4 C44 24.4 46.5 23.6 50 20.4 C53.5 23.6 56 24.4 58.5 24.4 '
            'C63.4 24.4 66.8 21.2 66.8 17 C66.8 12.8 63.4 9.6 58.5 9.6 C56 9.6 53.5 10.4 50 13.6 Z '
            'M46.2 17 C44.4 15.3 43 14.6 41.6 14.6 C39.9 14.6 38.6 15.7 38.6 17 C38.6 18.3 39.9 19.4 41.6 19.4 '
            'C43 19.4 44.4 18.7 46.2 17 Z M53.8 17 C55.6 18.7 57 19.4 58.4 19.4 C60.1 19.4 61.4 18.3 61.4 17 '
            'C61.4 15.7 60.1 14.6 58.4 14.6 C57 14.6 55.6 15.3 53.8 17 Z"/>')
STEM = '<rect x="47.5" y="21" width="5" height="6"/>'
INF_SHAPES = BASE + BODY + COLLAR + CUP + STEM + INFINITY
INF_LINES = K_LINES

# kind: (template kind, its shapes, its lines, new shapes, new lines)
KINGS = {
    'E': ('K', K_SHAPES, K_LINES, EYE_SHAPES, EYE_LINES),
    'S': ('K', K_SHAPES, K_LINES, SWAP_SHAPES, SWAP_LINES),
    'H': ('P', P_SHAPES, P_LINES, PAWN_SHAPES, PAWN_LINES),
    'I': ('K', K_SHAPES, K_LINES, INF_SHAPES, INF_LINES),
}
NAMES = {'E': 'The Eye', 'S': 'Swap king', 'H': 'Pawn king', 'I': 'Infinity king'}


def redraw(svg, side, src, dst, old_shapes, old_lines, new_shapes, new_lines):
    if old_lines not in svg or old_shapes not in svg:
        raise ValueError('template is not drawn from the classic shapes')
    svg = re.sub(r'<metadata>.*?</metadata>', '', svg, flags=re.S)  # provenance of the old file
    svg = svg.replace(old_shapes, new_shapes).replace(old_lines, new_lines)
    # Keep ids apart from the template's (e.g. clip path "blbKc" becomes "blbEc").
    for i in set(re.findall(r'id="([^"]+)"', svg)):
        if side + src in i:
            j = i.replace(side + src, side + dst)
            svg = svg.replace(f'id="{i}"', f'id="{j}"').replace(f'#{i})', f'#{j})').replace(f'"#{i}"', f'"#{j}"')
    return svg


def main():
    sets = [''] + sorted(d for d in os.listdir(PIECES) if os.path.isdir(os.path.join(PIECES, d)))
    rows = []
    for s in sets:
        sides = sorted({f[0] for f in os.listdir(os.path.join(PIECES, s)) if re.fullmatch(r'[a-z]K\.svg', f)})
        out = os.path.join(OUT, s or 'classic')
        os.makedirs(out, exist_ok=True)
        for side in sides:
            row = []
            for kind, (src, *args) in KINGS.items():
                svg = open(os.path.join(PIECES, s, f'{side}{src}.svg')).read()
                open(os.path.join(out, f'{side}{kind}.svg'), 'w').write(redraw(svg, side, src, kind, *args))
                row.append(f'{s or "classic"}/{side}{kind}.svg')
            rows.append((s or 'classic', side, row))
    html = ['<!doctype html><meta charset="utf-8"><title>Kings</title><style>'
            'body{margin:0;padding:16px;font:13px "IBM Plex Mono",monospace;background:#fffabd;color:#050505}'
            '@media (prefers-color-scheme:dark){body{background:#050505;color:#fffabd}}'
            'table{border-collapse:collapse}td{padding:2px 8px}img{width:56px;height:56px;display:block}'
            '.sq{background:#e9e3a6;padding:4px}.sq.d{background:#8f8a5a}</style><table>'
            '<tr><td></td>' + ''.join(f'<td>{NAMES[k]}</td>' for k in KINGS) + '</tr>']
    for s, side, row in rows:
        cells = ''.join(f'<td class="sq{" d" if n % 2 else ""}"><img src="{p}" alt=""></td>' for n, p in enumerate(row))
        html.append(f'<tr><td>{s} {side}</td>{cells}</tr>')
    open(os.path.join(OUT, 'index.html'), 'w').write(''.join(html) + '</table>')


if __name__ == '__main__':
    main()
