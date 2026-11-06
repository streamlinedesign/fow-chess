package main

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	"image/png"
	"strings"
	"sync"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// The link preview pictures: a 1200 by 630 card with a board at the left and
// the page's title and a few lines at the right, in the light theme's paper
// and ink. The piece and king PNGs are drawn from the web app's SVGs by
// `make share-assets`; the face is Geist, as on the site.
//
//go:embed og
var ogFiles embed.FS

const (
	ogW, ogH = 1200, 630
	ogSquare = 64
	ogBoardX = 59
	ogBoardY = (ogH - 8*ogSquare) / 2
	ogTextX  = ogBoardX + 8*ogSquare + 64
	ogTextW  = ogW - ogTextX - 56
)

var (
	ogPaper   = color.RGBA{0xff, 0xfa, 0xbd, 0xff}
	ogInk     = color.RGBA{0x05, 0x05, 0x05, 0xff}
	ogAccent  = color.RGBA{0x50, 0x50, 0x50, 0xff}
	ogLight   = color.RGBA{0xfb, 0xf5, 0xbf, 0xff}
	ogDark    = color.RGBA{0xae, 0xab, 0x8c, 0xff}
	ogFogL    = color.RGBA{0x3a, 0x3a, 0x33, 0xff}
	ogFogD    = color.RGBA{0x2b, 0x2b, 0x26, 0xff}
	ogHillDot = color.RGBA{0xc2, 0x41, 0x0c, 0xff}
)

// ogAssets are loaded once, on the first picture.
var ogAssets = sync.OnceValue(func() (a struct {
	pieces         map[byte]image.Image
	king           image.Image
	bold, regular  *opentype.Font
	brand          font.Face
	title, titleSm font.Face
	line, footnote font.Face
}) {
	load := func(name string) image.Image {
		f, err := ogFiles.Open("og/" + name)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		img, err := png.Decode(f)
		if err != nil {
			panic(err)
		}
		return img
	}
	scaled := func(img image.Image, size int) image.Image {
		dst := image.NewRGBA(image.Rect(0, 0, size, size))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
		return dst
	}
	a.pieces = map[byte]image.Image{}
	for _, k := range "KQRBNPJX" {
		a.pieces[byte(k)] = scaled(load("pieces/w"+string(k)+".png"), ogSquare)
		a.pieces[byte(k)+'a'-'A'] = scaled(load("pieces/b"+string(k)+".png"), ogSquare)
	}
	a.king = scaled(load("king.png"), 64)
	parse := func(name string) *opentype.Font {
		b, err := ogFiles.ReadFile("og/" + name)
		if err != nil {
			panic(err)
		}
		f, err := opentype.Parse(b)
		if err != nil {
			panic(err)
		}
		return f
	}
	face := func(f *opentype.Font, size float64) font.Face {
		fc, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
		if err != nil {
			panic(err)
		}
		return fc
	}
	a.bold, a.regular = parse("Geist-SemiBold.ttf"), parse("Geist-Regular.ttf")
	a.brand = face(a.bold, 40)
	a.title = face(a.bold, 58)
	a.titleSm = face(a.bold, 46)
	a.line = face(a.regular, 30)
	a.footnote = face(a.regular, 26)
	return a
})

// renderCard draws a card as a PNG. board is 64 characters from a1 to h8, as
// boardView writes them: FEN letters, '.' for empty and '?' for fog. J and X
// are the jester pawn and the prince.
func renderCard(c card, host string) []byte {
	a := ogAssets()
	img := image.NewRGBA(image.Rect(0, 0, ogW, ogH))
	fill(img, img.Bounds(), ogPaper)

	// The board, White at the bottom, in a frame of ink.
	fill(img, image.Rect(ogBoardX-4, ogBoardY-4, ogBoardX+8*ogSquare+4, ogBoardY+8*ogSquare+4), ogInk)
	for i := 0; i < 64; i++ {
		file, rank := i%8, i/8
		x, y := ogBoardX+file*ogSquare, ogBoardY+(7-rank)*ogSquare
		sq := image.Rect(x, y, x+ogSquare, y+ogSquare)
		light := (file+rank)%2 == 1
		var ch byte = '.'
		if i < len(c.Board) {
			ch = c.Board[i]
		}
		switch {
		case ch == '?' && light:
			fill(img, sq, ogFogL)
		case ch == '?':
			fill(img, sq, ogFogD)
		case light:
			fill(img, sq, ogLight)
		default:
			fill(img, sq, ogDark)
		}
		if c.Hill && (file == 3 || file == 4) && (rank == 3 || rank == 4) {
			// The hill: a dot in the corner nearest the centre.
			cx, cy := x+ogSquare/2, y+ogSquare/2
			if file == 3 {
				cx += ogSquare/2 - 10
			} else {
				cx -= ogSquare/2 - 10
			}
			if rank == 4 {
				cy += ogSquare/2 - 10
			} else {
				cy -= ogSquare/2 - 10
			}
			fill(img, image.Rect(cx-4, cy-4, cx+4, cy+4), ogHillDot)
		}
		if p := a.pieces[ch]; p != nil {
			draw.Draw(img, sq, p, image.Point{}, draw.Over)
		}
	}

	// The name with the fallen king, as in the header.
	y := ogBoardY
	draw.Draw(img, image.Rect(ogTextX-8, y-6, ogTextX+56, y+58), a.king, image.Point{}, draw.Over)
	text(img, a.brand, ogInk, ogTextX+66, y+42, "Eat the King")
	y += 120

	// The title in up to three lines, smaller when it's long.
	face, lh := a.title, 66
	lines := wrap(a.title, c.Title, ogTextW)
	if len(lines) > 2 {
		face, lh = a.titleSm, 54
		lines = wrap(a.titleSm, c.Title, ogTextW)
	}
	for _, l := range clipLines(lines, 3) {
		text(img, face, ogInk, ogTextX, y, l)
		y += lh
	}
	y += 14
	for _, l := range c.Lines {
		for _, w := range clipLines(wrap(a.line, l, ogTextW), 2) {
			if y > ogBoardY+8*ogSquare-60 {
				break
			}
			text(img, a.line, ogAccent, ogTextX, y, w)
			y += 40
		}
		y += 6
	}
	if host != "" {
		text(img, a.footnote, ogInk, ogTextX, ogBoardY+8*ogSquare-4, host)
	}

	var buf bytes.Buffer
	(&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&buf, img)
	return buf.Bytes()
}

func fill(img *image.RGBA, r image.Rectangle, c color.Color) {
	draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
}

// text draws s with its baseline at y.
func text(img *image.RGBA, f font.Face, c color.Color, x, y int, s string) {
	d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: f, Dot: fixed.P(x, y)}
	d.DrawString(s)
}

// wrap breaks s into lines no wider than w.
func wrap(f font.Face, s string, w int) []string {
	var lines []string
	cur := ""
	for _, word := range strings.Fields(s) {
		next := word
		if cur != "" {
			next = cur + " " + word
		}
		if cur != "" && font.MeasureString(f, next).Ceil() > w {
			lines = append(lines, cur)
			next = word
		}
		cur = next
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// clipLines keeps the first n lines, ending the last kept one with an ellipsis if any were cut.
func clipLines(lines []string, n int) []string {
	if len(lines) <= n {
		return lines
	}
	out := append([]string{}, lines[:n]...)
	out[n-1] = strings.TrimRight(out[n-1], " ,.·") + "…"
	return out
}
