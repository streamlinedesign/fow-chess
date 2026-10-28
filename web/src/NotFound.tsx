import { pieceImg } from './position'
import { navigate } from './router'

/** Any address the site doesn't know. */
export function NotFound() {
  return (
    <main className="mx-auto w-full max-w-[1184px] flex-1 px-4 pt-6 pb-16 sm:px-8">
      <section className="nf">
        <div className="nf-board grid aspect-square grid-cols-8 overflow-hidden rounded-[3px]" aria-hidden>
          {Array.from({ length: 64 }, (_, i) => (
            // Only the king on e1 can be seen; the rest is fog.
            <div key={i} className="sq" data-light={(Math.floor(i / 8) + i) % 2 === 0} data-fog={i !== 60 || undefined}>
              {i === 60 && <img className="piece" src={pieceImg('K')} alt="" />}
            </div>
          ))}
        </div>
        <div>
          <div className="nf-sq">e9</div>
          <h1>There's no such square</h1>
          <p>The page you were looking for doesn't exist, or the link was mistyped.</p>
          <a
            href="/"
            onClick={(e) => {
              e.preventDefault()
              navigate('/')
            }}
            className="mt-6 inline-flex h-12 items-center rounded-md bg-fg px-6 text-[17px] font-semibold text-bg hover:opacity-85"
          >
            Back home
          </a>
        </div>
      </section>
    </main>
  )
}
