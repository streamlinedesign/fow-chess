/** Small line icons on a 15 by 15 grid, drawn in the current text colour. */
const paths = {
  bell: (
    <>
      <path
        d="M8.6 1.56a1.1 1.1 0 1 0-2.2 0v.33A4.21 4.21 0 0 0 3.29 6v2.58c0 .56-.27 1.2-.78 1.73l-.53.56a.5.5 0 0 0 .36.85h10.32a.5.5 0 0 0 .36-.85l-.53-.56c-.51-.53-.78-1.17-.78-1.73V6A4.21 4.21 0 0 0 8.6 1.89v-.33Z"
        strokeLinejoin="round"
      />
      <path d="M6 12.8a1.5 1.5 0 0 0 3 0" strokeLinecap="round" />
    </>
  ),
  user: (
    <>
      <circle cx="7.5" cy="5" r="2.5" />
      <path d="M2.5 13c.6-2.5 2.6-4 5-4s4.4 1.5 5 4" strokeLinecap="round" />
    </>
  ),
  people: (
    <>
      <circle cx="5.5" cy="5" r="2.2" />
      <path
        d="M1.5 12.5c.5-2.2 2-3.5 4-3.5s3.5 1.3 4 3.5M10 3a2.2 2.2 0 0 1 0 4.2M11.5 9.3c1 .5 1.7 1.6 2 3.2"
        strokeLinecap="round"
      />
    </>
  ),
  chat: (
    <path
      d="M2 3.5A1.5 1.5 0 0 1 3.5 2h8A1.5 1.5 0 0 1 13 3.5v5.5a1.5 1.5 0 0 1-1.5 1.5H6l-3 2.5v-2.5h.5A1.5 1.5 0 0 1 2 9V3.5Z"
      strokeLinejoin="round"
    />
  ),
  gear: (
    <>
      <circle cx="7.5" cy="7.5" r="2" />
      <path
        d="M7.5 1.5v1.6M7.5 11.9v1.6M13.5 7.5h-1.6M3.1 7.5H1.5M11.74 3.26l-1.13 1.13M4.39 10.61l-1.13 1.13M11.74 11.74l-1.13-1.13M4.39 4.39 3.26 3.26"
        strokeLinecap="round"
      />
    </>
  ),
  out: (
    <path
      d="M6 2H3.5A1.5 1.5 0 0 0 2 3.5v8A1.5 1.5 0 0 0 3.5 13H6M10 10.5l3-3-3-3M13 7.5H5.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  ),
  search: (
    <>
      <circle cx="6.5" cy="6.5" r="4.5" />
      <path d="m10 10 3.5 3.5" strokeLinecap="round" />
    </>
  ),
  back: <path d="M9 3 4.5 7.5 9 12" strokeLinecap="round" strokeLinejoin="round" />,
  swords: (
    <path
      d="M2 2l6.5 6.5M2 2h3M2 2v3M13 2 6.5 8.5M13 2h-3M13 2v3M4 9.5 2 11.5l1.5 1.5 2-2M11 9.5l2 2-1.5 1.5-2-2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  ),
  check: <path d="m3 7.8 3 3 6-6.6" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" />,
  plus: <path d="M7.5 2.5v10M2.5 7.5h10" strokeLinecap="round" />,
  send: <path d="M2 7.5h9.5M8 3.5l4 4-4 4" strokeLinecap="round" strokeLinejoin="round" />,
  trend: <path d="M1.5 11 5.5 7l2.5 2.5L13.5 4M10 4h3.5v3.5" strokeLinecap="round" strokeLinejoin="round" />,
}

export type IconName = keyof typeof paths | 'dots'

export function Icon({ name, size = 16, className = '' }: { name: IconName; size?: number; className?: string }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 15 15"
      fill="none"
      stroke="currentColor"
      aria-hidden
      className={className}
    >
      {name === 'dots' ? (
        <g fill="currentColor" stroke="none">
          <circle cx="3" cy="7.5" r="1.1" />
          <circle cx="7.5" cy="7.5" r="1.1" />
          <circle cx="12" cy="7.5" r="1.1" />
        </g>
      ) : (
        paths[name]
      )}
    </svg>
  )
}
