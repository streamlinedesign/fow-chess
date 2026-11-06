import { memo } from "react";

/**
 * The smoke over a board: one blurred layer of the hidden squares, in each
 * square's fog colour, so the fog reads as a single cloud with soft edges
 * instead of separate dark tiles. A second, sharp layer carries the grain.
 * Drawn only while the smoke style is on (index.css hides it for dusk).
 *
 * `cells` lists every grid cell in reading order: a colour for a hidden
 * square, null for one you can see or a cell off the board.
 */
export const FogLayer = memo(function FogLayer({
  cells,
  size,
}: {
  cells: (string | null)[];
  size: number;
}) {
  if (!cells.some(Boolean)) return null;
  const grid = { gridTemplate: `repeat(${size}, 1fr) / repeat(${size}, 1fr)` };
  // Each grain cell shows its own part of one board-sized texture, so there are no seams.
  const at = (i: number) =>
    `${((i % size) / (size - 1)) * 100}% ${(Math.floor(i / size) / (size - 1)) * 100}%`;
  return (
    <>
      <div className="fog-smoke" style={grid} aria-hidden>
        {cells.map((c, i) => (
          <i
            key={i}
            style={c ? ({ "--fog-tint": c } as React.CSSProperties) : undefined}
          />
        ))}
      </div>
      <div
        className="fog-grain"
        style={{ ...grid, "--fog-n": size } as React.CSSProperties}
        aria-hidden
      >
        {cells.map((c, i) => (
          <i
            key={i}
            style={c ? { backgroundPosition: at(i) } : undefined}
            data-on={c ? true : undefined}
          />
        ))}
      </div>
    </>
  );
});
