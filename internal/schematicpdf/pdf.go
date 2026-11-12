package schematicpdf

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/goregular"
)

type pdfPainter struct {
	doc       *fpdf.Fpdf
	ctx       context.Context
	clipped   bool
	current   style
	styles    []style
	grayscale bool
}

func newPDF(ctx context.Context, page Page, scale, x, y float64) (*pdfPainter, error) {
	doc := fpdf.NewCustom(&fpdf.InitType{UnitStr: "pt", Size: fpdf.SizeType{Wd: page.Width, Ht: page.Height}})
	doc.SetAutoPageBreak(false, 0)
	doc.SetTitle("ELab schematic", true)
	doc.SetCreator("ELab", true)
	doc.SetSubject(page.Metadata, true)
	doc.SetCatalogSort(true)
	// The capture metadata records source identity; wall time is not needed.
	epoch := time.Unix(0, 0).UTC()
	doc.SetCreationDate(epoch)
	doc.SetModificationDate(epoch)
	doc.AddUTF8FontFromBytes("Go", "", goregular.TTF)
	p := &pdfPainter{doc: doc, ctx: ctx, grayscale: page.Grayscale}
	return p, p.beginPage(page, scale, x, y)
}

func (p *pdfPainter) beginPage(page Page, scale, x, y float64) error {
	d := p.doc
	d.AddPage()
	d.SetLineCapStyle("round")
	d.SetLineJoinStyle("round")
	d.SetFillColor(255, 255, 255)
	d.Rect(0, 0, page.Width, page.Height, "F")
	if page.Tile {
		d.SetTextColor(0, 0, 0)
		d.SetFont("Go", "", 8.5)
		d.Text(page.Margin, page.Height-page.Margin-5, page.Footer)
		d.ClipRect(page.Margin, page.Margin, page.Width-2*page.Margin, page.Height-2*page.Margin-tileFooterHeight, false)
	}
	p.clipped = page.Tile
	d.TransformBegin()
	d.TransformTranslate(x, y)
	d.TransformScaleXY(scale*100, 0, 0)
	return d.Error()
}

func (p *pdfPainter) endPage() {
	p.doc.TransformEnd()
	if p.clipped {
		p.doc.ClipEnd()
	}
}

func (p *pdfPainter) nextPage(page Page, scale, x, y float64) error {
	p.endPage()
	return p.beginPage(page, scale, x, y)
}

func (p *pdfPainter) push() {
	p.styles = append(p.styles, p.current)
	p.doc.TransformBegin()
}
func (p *pdfPainter) pop() {
	p.current = p.styles[len(p.styles)-1]
	p.styles = p.styles[:len(p.styles)-1]
	p.doc.TransformEnd()
}
func (p *pdfPainter) transform(x, y, rotation float64) {
	p.doc.TransformTranslate(x, y)
	p.doc.TransformRotate(-rotation, 0, 0)
}

func (p *pdfPainter) style(s style) { p.current = s }

func (p *pdfPainter) draw(path drawing) {
	s, d := p.current, p.doc
	mode := ""
	if s.fill != "none" {
		c := rgb(s.fill, p.grayscale)
		d.SetFillColor(int(c.R), int(c.G), int(c.B))
		mode = "F"
	}
	if s.stroke != "none" && s.width > 0 {
		c := rgb(s.stroke, p.grayscale)
		d.SetDrawColor(int(c.R), int(c.G), int(c.B))
		d.SetLineWidth(s.width)
		mode += "D"
	}
	if mode == "" || len(path) == 0 {
		return
	}
	var prev, start point
	for _, cmd := range path {
		a, b, c := cmd.p, cmd.q, cmd.r
		switch cmd.op {
		case 'M':
			d.MoveTo(a.x, a.y)
			prev, start = a, a
		case 'L':
			d.LineTo(a.x, a.y)
			prev = a
		case 'Q':
			u := prev.add(a.sub(prev).mul(2. / 3))
			v := b.add(a.sub(b).mul(2. / 3))
			d.CurveBezierCubicTo(u.x, u.y, v.x, v.y, b.x, b.y)
			prev = b
		case 'C':
			d.CurveBezierCubicTo(a.x, a.y, b.x, b.y, c.x, c.y)
			prev = c
		case 'Z':
			d.ClosePath()
			prev = start
		}
	}
	d.DrawPath(mode)
}

func (p *pdfPainter) text(x, y, size float64, value string) error {
	if p.current.fill == "none" || size <= 0 || value == "" {
		return nil
	}
	value = fontText(value)
	d := p.doc
	c := rgb(p.current.fill, p.grayscale)
	d.SetTextColor(int(c.R), int(c.G), int(c.B))
	d.SetFont("Go", "", size)
	if p.current.anchor == "end" {
		x -= d.GetStringWidth(value)
	}
	d.Text(x, y, value)
	return d.Error()
}

func (p *pdfPainter) finish() ([]byte, error) {
	p.endPage()
	out := outputBuffer{ctx: p.ctx}
	if err := p.doc.Output(&out); err != nil {
		return nil, fmt.Errorf("encode schematic PDF: %w", err)
	}
	return out.Bytes(), nil
}
