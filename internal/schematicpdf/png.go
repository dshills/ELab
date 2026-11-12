package schematicpdf

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

var labelFont = func() *sfnt.Font {
	f, err := sfnt.Parse(goregular.TTF)
	if err != nil {
		panic(err)
	} // compiled-in, known font
	return f
}()

func fontText(value string) string {
	return strings.Map(func(r rune) rune {
		if g, err := labelFont.GlyphIndex(nil, r); err != nil || g == 0 || r > 0xffff {
			return '?'
		}
		return r
	}, value)
}

// pose maps drawing coordinates to pixels. The two page axes may have
// slightly different scales because the bitmap dimensions are rounded up.
type pose struct{ origin, xAxis, yAxis point }

func (m pose) vector(p point) point { return m.xAxis.mul(p.x).add(m.yAxis.mul(p.y)) }
func (m pose) at(p point) point     { return m.origin.add(m.vector(p)) }
func (m pose) placed(x, y, rotation float64) pose {
	s, c := math.Sincos(rotation * math.Pi / 180)
	return pose{m.at(point{x, y}), m.vector(point{c, s}), m.vector(point{-s, c})}
}

type pngState struct {
	pose
	style
}
type pngPainter struct {
	ctx       context.Context
	im        *image.RGBA
	clip      image.Rectangle
	current   pngState
	stack     []pngState
	mask      vector.Rasterizer
	grayscale bool
	dpi       int
	metadata  string
}

func newPNG(ctx context.Context, page Page, scale, x, y float64) (*pngPainter, error) {
	w, h := rasterDimensions(page)
	im := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	draw.Draw(im, im.Bounds(), image.White, image.Point{}, draw.Src)
	dx, dy := w/page.Width, h/page.Height
	p := &pngPainter{ctx: ctx, im: im, clip: im.Bounds(), grayscale: page.Grayscale, dpi: page.RasterDPI, metadata: page.Metadata}
	p.current.pose = pose{xAxis: point{dx, 0}, yAxis: point{0, dy}}
	if page.Tile {
		p.current.style = style{fill: "#000000"}
		if err := p.text(page.Margin, page.Height-page.Margin-5, 8.5, page.Footer); err != nil {
			return nil, err
		}
		p.clip = image.Rect(int(math.Ceil(page.Margin*dx)), int(math.Ceil(page.Margin*dy)),
			int(math.Floor((page.Width-page.Margin)*dx)), int(math.Floor((page.Height-page.Margin-tileFooterHeight)*dy)))
	}
	p.current.pose = pose{point{x * dx, y * dy}, point{scale * dx, 0}, point{0, scale * dy}}
	return p, nil
}

func (*pngPainter) nextPage(Page, float64, float64, float64) error {
	return fmt.Errorf("PNG contains exactly one selected page")
}
func (p *pngPainter) push() { p.stack = append(p.stack, p.current) }
func (p *pngPainter) pop()  { p.current = p.stack[len(p.stack)-1]; p.stack = p.stack[:len(p.stack)-1] }
func (p *pngPainter) transform(x, y, rotation float64) {
	p.current.pose = p.current.placed(x, y, rotation)
}
func (p *pngPainter) style(s style) { p.current.style = s }

func (p *pngPainter) draw(d drawing) {
	if p.current.fill != "none" {
		p.fill(d, rgb(p.current.fill, p.grayscale))
	}
	if p.current.stroke != "none" && p.current.width > 0 {
		// Flatten curves to within a fraction of a pixel before building the
		// round caps and joins. Generate the stroke in drawing coordinates.
		m := p.current.pose
		density := max(math.Hypot(m.xAxis.x, m.xAxis.y), math.Hypot(m.yAxis.x, m.yAxis.y))
		p.fill(roundStroke(d, p.current.width/2, 0.25/density), rgb(p.current.stroke, p.grayscale))
	}
}

func (p *pngPainter) fill(d drawing, c color.RGBA) {
	if len(d) == 0 || p.ctx.Err() != nil || p.clip.Empty() {
		return
	}
	m := p.current.pose
	left, top, right, bottom := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	include := func(a point) {
		a = m.at(a)
		left, top, right, bottom = min(left, a.x), min(top, a.y), max(right, a.x), max(bottom, a.y)
	}
	for _, s := range d {
		if s.op == 'Z' {
			continue
		}
		include(s.p)
		if s.op == 'Q' || s.op == 'C' {
			include(s.q)
		}
		if s.op == 'C' {
			include(s.r)
		}
	}
	loX, loY := max(float64(p.clip.Min.X), math.Floor(left)), max(float64(p.clip.Min.Y), math.Floor(top))
	hiX, hiY := min(float64(p.clip.Max.X), math.Ceil(right)), min(float64(p.clip.Max.Y), math.Ceil(bottom))
	if !(loX < hiX && loY < hiY) {
		return
	}
	bounds := image.Rect(int(loX), int(loY), int(hiX), int(hiY))
	p.mask.Reset(bounds.Dx(), bounds.Dy())
	pos := func(a point) (float32, float32) { a = m.at(a); return float32(a.x - loX), float32(a.y - loY) }
	for _, s := range d {
		switch s.op {
		case 'M':
			p.mask.ClosePath()
			p.mask.MoveTo(pos(s.p))
		case 'L':
			p.mask.LineTo(pos(s.p))
		case 'Q':
			x, y := pos(s.p)
			ex, ey := pos(s.q)
			p.mask.QuadTo(x, y, ex, ey)
		case 'C':
			x, y := pos(s.p)
			u, v := pos(s.q)
			ex, ey := pos(s.r)
			p.mask.CubeTo(x, y, u, v, ex, ey)
		case 'Z':
			p.mask.ClosePath()
		}
	}
	p.mask.ClosePath()
	p.mask.Draw(p.im, bounds, image.NewUniform(c), image.Point{})
}

func roundStroke(d drawing, r, tolerance float64) drawing {
	var out drawing
	var last, start point
	line := func(to point) {
		delta := to.sub(last)
		length := math.Hypot(delta.x, delta.y)
		if length > 0 {
			n := point{-delta.y * r / length, delta.x * r / length}
			out.move(last.add(n))
			out.line(last.sub(n))
			out.line(to.sub(n))
			out.line(to.add(n))
			out.end()
			out = append(out, circle(to.x, to.y, r)...)
		}
		last = to
	}
	// Subdivide Bezier curves until the control polygon is close to the
	// straight chord. The depth cap bounds work for any generated geometry.
	var curve func(point, point, point, point, int)
	curve = func(a, b, c, z point, depth int) {
		d1 := b.sub(a.add(z.sub(a).mul(1. / 3)))
		d2 := c.sub(a.add(z.sub(a).mul(2. / 3)))
		if depth >= 10 || max(math.Hypot(d1.x, d1.y), math.Hypot(d2.x, d2.y)) <= tolerance {
			line(z)
			return
		}
		ab, bc, cz := a.add(b).mul(.5), b.add(c).mul(.5), c.add(z).mul(.5)
		u, v := ab.add(bc).mul(.5), bc.add(cz).mul(.5)
		mid := u.add(v).mul(.5)
		curve(a, ab, u, mid, depth+1)
		curve(mid, v, cz, z, depth+1)
	}
	for _, s := range d {
		switch s.op {
		case 'M':
			last, start = s.p, s.p
			out = append(out, circle(last.x, last.y, r)...)
		case 'L':
			line(s.p)
		case 'Q':
			a := last
			curve(a, a.add(s.p.sub(a).mul(2./3)), s.q.add(s.p.sub(s.q).mul(2./3)), s.q, 0)
		case 'C':
			curve(last, s.p, s.q, s.r, 0)
		case 'Z':
			line(start)
		}
	}
	return out
}

func (p *pngPainter) text(x, y, size float64, value string) error {
	if p.current.fill == "none" || size <= 0 || value == "" {
		return nil
	}
	value = fontText(value)
	var d drawing
	var b sfnt.Buffer
	var width float64
	ppem := fixed.I(1024)
	factor := size / float64(ppem)
	for _, r := range value {
		g, err := labelFont.GlyphIndex(&b, r)
		if err != nil {
			return err
		}
		advance, err := labelFont.GlyphAdvance(&b, g, ppem, font.HintingNone)
		if err != nil {
			return err
		}
		segments, err := labelFont.LoadGlyph(&b, g, ppem, nil)
		if err != nil {
			return err
		}
		at := func(q fixed.Point26_6) point { return point{width + float64(q.X)*factor, float64(q.Y) * factor} }
		for _, s := range segments {
			a, c, z := at(s.Args[0]), at(s.Args[1]), at(s.Args[2])
			switch s.Op {
			case sfnt.SegmentOpMoveTo:
				if len(d) != 0 {
					d.end()
				}
				d.move(a)
			case sfnt.SegmentOpLineTo:
				d.line(a)
			case sfnt.SegmentOpQuadTo:
				d.quad(a, c)
			case sfnt.SegmentOpCubeTo:
				d.cubic(a, c, z)
			}
		}
		width += float64(advance) * factor
	}
	if len(d) > 0 {
		d.end()
		if p.current.anchor == "end" {
			x -= width
		}
		before := p.current.pose
		p.current.pose = before.placed(x, y, 0)
		p.fill(d, rgb(p.current.fill, p.grayscale))
		p.current.pose = before
	}
	return nil
}

func (p *pngPainter) finish() ([]byte, error) {
	out := outputBuffer{ctx: p.ctx}
	if err := png.Encode(&out, p.im); err != nil {
		return nil, fmt.Errorf("encode schematic PNG: %w", err)
	}
	// Add physical resolution and a UTF-8 description after IHDR and before IDAT.
	var resolution [9]byte
	ppm := uint32(math.Round(float64(p.dpi) / 0.0254))
	binary.BigEndian.PutUint32(resolution[:4], ppm)
	binary.BigEndian.PutUint32(resolution[4:8], ppm)
	resolution[8] = 1
	var header bytes.Buffer
	writePNGChunk(&header, "pHYs", resolution[:])
	if p.metadata != "" {
		writePNGChunk(&header, "iTXt", []byte("Description\x00\x00\x00\x00\x00"+p.metadata))
	}
	const ihdrEnd = 8 + 12 + 13
	data := out.Bytes()
	if len(data)+header.Len() > maxOutput {
		return nil, fmt.Errorf("schematic output exceeds export bounds")
	}
	return bytes.Join([][]byte{data[:ihdrEnd], header.Bytes(), data[ihdrEnd:]}, nil), nil
}

func writePNGChunk(out *bytes.Buffer, kind string, data []byte) {
	_ = binary.Write(out, binary.BigEndian, uint32(len(data)))
	start := out.Len()
	out.WriteString(kind)
	out.Write(data)
	_ = binary.Write(out, binary.BigEndian, crc32.ChecksumIEEE(out.Bytes()[start:]))
}
