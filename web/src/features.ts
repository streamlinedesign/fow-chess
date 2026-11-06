/** Parts of the site that are built but switched off for now. */

/** Tournaments (arenas): hidden from the header, the menu and the home page, and /tournaments is a 404. The server still runs them. */
export const TOURNAMENTS = false

/** The desktop app's downloads: the header link, the menu item, the home page strip and /download (a 404 while off). Back on for 1.0, once a release has the installers. */
export const DOWNLOADS = false

/** The day the beta ends and 1.0 ships (home page and the rules page's plans). */
export const RELEASE = '2027-02-02'

/** RELEASE as a date in the reader's language, like "2 February 2027". */
export function releaseDate(locale: string) {
  return new Intl.DateTimeFormat(locale, { day: 'numeric', month: 'long', year: 'numeric' }).format(
    new Date(`${RELEASE}T12:00:00`),
  )
}
