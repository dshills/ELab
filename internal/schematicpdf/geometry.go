package schematicpdf

import (
	"fmt"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type point struct{ x, y float64 }

func (p point) add(q point) point   { return point{p.x + q.x, p.y + q.y} }
func (p point) sub(q point) point   { return point{p.x - q.x, p.y - q.y} }
func (p point) mul(s float64) point { return point{p.x * s, p.y * s} }

type segment struct {
	op      byte
	p, q, r point
}

type drawing []segment

func (d *drawing) move(p point)        { *d = append(*d, segment{op: 'M', p: p}) }
func (d *drawing) line(p point)        { *d = append(*d, segment{op: 'L', p: p}) }
func (d *drawing) quad(p, q point)     { *d = append(*d, segment{op: 'Q', p: p, q: q}) }
func (d *drawing) cubic(p, q, r point) { *d = append(*d, segment{op: 'C', p: p, q: q, r: r}) }
func (d *drawing) end()                { *d = append(*d, segment{op: 'Z'}) }

func rectangle(x, y, w, h float64) drawing {
	return drawing{{op: 'M', p: point{x, y}}, {op: 'L', p: point{x + w, y}},
		{op: 'L', p: point{x + w, y + h}}, {op: 'L', p: point{x, y + h}}, {op: 'Z'}}
}

func circle(x, y, r float64) drawing {
	k := r * 0.5522847498307936 // cubic approximation of a quarter circle
	return drawing{
		{op: 'M', p: point{x + r, y}},
		{op: 'C', p: point{x + r, y + k}, q: point{x + k, y + r}, r: point{x, y + r}},
		{op: 'C', p: point{x - k, y + r}, q: point{x - r, y + k}, r: point{x - r, y}},
		{op: 'C', p: point{x - r, y - k}, q: point{x - k, y - r}, r: point{x, y - r}},
		{op: 'C', p: point{x + k, y - r}, q: point{x + r, y - k}, r: point{x + r, y}},
		{op: 'Z'},
	}
}

var pathTokens = regexp.MustCompile(`[MLHVQZ]|[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?`)

func parsePath(data string) (drawing, error) {
	tokens := pathTokens.FindAllString(data, -1)
	if strings.TrimSpace(strings.ReplaceAll(pathTokens.ReplaceAllString(data, ""), ",", "")) != "" {
		return nil, fmt.Errorf("unsupported generated path syntax")
	}
	var d drawing
	var p, start point
	for len(tokens) > 0 {
		op := tokens[0]
		count, ok := map[string]int{"M": 2, "L": 2, "H": 1, "V": 1, "Q": 4, "Z": 0}[op]
		if !ok || len(d) == 0 && op != "M" {
			return nil, fmt.Errorf("unsupported generated path command")
		}
		if len(tokens) < 1+count {
			return nil, fmt.Errorf("incomplete generated path")
		}
		var v [4]float64
		for k := range count {
			n, err := strconv.ParseFloat(tokens[1+k], 64)
			if err != nil || !isFinite(n) {
				return nil, fmt.Errorf("invalid generated path coordinate")
			}
			v[k] = n
		}
		tokens = tokens[1+count:]
		switch op {
		case "M":
			p = point{v[0], v[1]}
			start = p
			d.move(p)
		case "L":
			p = point{v[0], v[1]}
			d.line(p)
		case "H":
			p.x = v[0]
			d.line(p)
		case "V":
			p.y = v[0]
			d.line(p)
		case "Q":
			p = point{v[2], v[3]}
			d.quad(point{v[0], v[1]}, p)
		case "Z":
			p = start
			d.end()
		}
	}
	return d, nil
}

func rgb(value string, gray bool) color.RGBA {
	var c color.RGBA
	c.A = 255
	if len(value) == 7 && value[0] == '#' {
		v, _ := strconv.ParseUint(value[1:], 16, 24)
		c.R, c.G, c.B = uint8(v>>16), uint8(v>>8), uint8(v)
	}
	if gray {
		y := uint8(math.Round(.2126*float64(c.R) + .7152*float64(c.G) + .0722*float64(c.B)))
		c.R, c.G, c.B = y, y, y
	}
	return c
}
