//go:build darwin && cgo

package schematicpdf

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreText -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>
#include <CoreText/CoreText.h>
#include <stdlib.h>

static CGContextRef elab_pdf_create(CFMutableDataRef data, double width, double height, const char *metadata) {
 CGDataConsumerRef consumer = CGDataConsumerCreateWithCFData(data);
 CGRect box = CGRectMake(0, 0, width, height);
 CFStringRef subject = CFStringCreateWithCString(NULL, metadata, kCFStringEncodingUTF8);
 const void *keys[] = {kCGPDFContextTitle, kCGPDFContextCreator, kCGPDFContextSubject};
 const void *values[] = {CFSTR("ELab schematic"), CFSTR("ELab CoreGraphics renderer"), subject};
 CFDictionaryRef info = CFDictionaryCreate(NULL, keys, values, 3, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
 CGContextRef context = CGPDFContextCreate(consumer, &box, info);
 CFRelease(info); CFRelease(subject); CGDataConsumerRelease(consumer);
 return context;
}
static int elab_pdf_text(CGContextRef context, const char *text, double size, double x, double y) {
 CFStringRef value = CFStringCreateWithCString(NULL, text, kCFStringEncodingUTF8);
 if (!value) return 0;
 CTFontRef font = CTFontCreateWithName(CFSTR("Helvetica"), size, NULL);
 const void *keys[] = {kCTFontAttributeName, kCTForegroundColorFromContextAttributeName};
 const void *values[] = {font, kCFBooleanTrue};
 CFDictionaryRef attributes = CFDictionaryCreate(NULL, keys, values, 2, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
 CFAttributedStringRef attributed = CFAttributedStringCreate(NULL, value, attributes);
 CTLineRef line = CTLineCreateWithAttributedString(attributed);
 CGContextSaveGState(context);
 CGContextSetTextMatrix(context, CGAffineTransformMakeScale(1, -1));
 CGContextSetTextPosition(context, x, y);
 CTLineDraw(line, context);
 CGContextRestoreGState(context);
 CFRelease(line); CFRelease(attributed); CFRelease(attributes); CFRelease(font); CFRelease(value);
 return 1;
}
*/
import "C"

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unsafe"
)

type quartz struct {
	ctx       C.CGContextRef
	data      C.CFMutableDataRef
	current   style
	grayscale bool
	finished  bool
}

func Supported() bool { return true }
func newPainter(page Page, scale, x, y float64) (painter, error) {
	data := C.CFDataCreateMutable(C.CFAllocatorRef(0), 0)
	metadata := C.CString(page.Metadata)
	defer C.free(unsafe.Pointer(metadata))
	ctx := C.elab_pdf_create(data, C.double(page.Width), C.double(page.Height), metadata)
	if ctx == 0 {
		C.CFRelease(C.CFTypeRef(data))
		return nil, fmt.Errorf("cannot create PDF context")
	}
	C.CGPDFContextBeginPage(ctx, C.CFDictionaryRef(0))
	C.CGContextSetRGBFillColor(ctx, 1, 1, 1, 1)
	C.CGContextFillRect(ctx, C.CGRectMake(0, 0, C.double(page.Width), C.double(page.Height)))
	C.CGContextTranslateCTM(ctx, C.double(x), C.double(page.Height-y))
	C.CGContextScaleCTM(ctx, C.double(scale), C.double(-scale))
	C.CGContextSetLineCap(ctx, C.kCGLineCapRound)
	C.CGContextSetLineJoin(ctx, C.kCGLineJoinRound)
	return &quartz{ctx: ctx, data: data, grayscale: page.Grayscale}, nil
}
func (p *quartz) push() { C.CGContextSaveGState(p.ctx) }
func (p *quartz) pop()  { C.CGContextRestoreGState(p.ctx) }
func (p *quartz) transform(x, y, rotation float64) {
	C.CGContextTranslateCTM(p.ctx, C.double(x), C.double(y))
	C.CGContextRotateCTM(p.ctx, C.double(rotation*math.Pi/180))
}
func (p *quartz) style(s style) {
	p.current = s
	rgb := func(color string) (float64, float64, float64) {
		if len(color) != 7 || color[0] != '#' {
			return 0, 0, 0
		}
		v, _ := strconv.ParseUint(color[1:], 16, 32)
		r, g, b := float64(v>>16)/255, float64((v>>8)&255)/255, float64(v&255)/255
		if p.grayscale {
			y := .2126*r + .7152*g + .0722*b
			return y, y, y
		}
		return r, g, b
	}
	r, g, b := rgb(s.fill)
	C.CGContextSetRGBFillColor(p.ctx, C.double(r), C.double(g), C.double(b), 1)
	r, g, b = rgb(s.stroke)
	C.CGContextSetRGBStrokeColor(p.ctx, C.double(r), C.double(g), C.double(b), 1)
	C.CGContextSetLineWidth(p.ctx, C.double(s.width))
}
func (p *quartz) draw() {
	fill, stroke := p.current.fill != "none", p.current.stroke != "none"
	switch {
	case fill && stroke:
		C.CGContextDrawPath(p.ctx, C.kCGPathFillStroke)
	case fill:
		C.CGContextFillPath(p.ctx)
	case stroke:
		C.CGContextStrokePath(p.ctx)
	default:
		C.CGContextBeginPath(p.ctx)
	}
}
func (p *quartz) rect(x, y, w, h float64) {
	C.CGContextBeginPath(p.ctx)
	C.CGContextAddRect(p.ctx, C.CGRectMake(C.double(x), C.double(y), C.double(w), C.double(h)))
	p.draw()
}
func (p *quartz) ellipse(x, y, r float64) {
	C.CGContextBeginPath(p.ctx)
	C.CGContextAddEllipseInRect(p.ctx, C.CGRectMake(C.double(x-r), C.double(y-r), C.double(2*r), C.double(2*r)))
	p.draw()
}
func (p *quartz) polyline(points []float64) {
	C.CGContextBeginPath(p.ctx)
	for i := 0; i+1 < len(points); i += 2 {
		if i == 0 {
			C.CGContextMoveToPoint(p.ctx, C.double(points[i]), C.double(points[i+1]))
		} else {
			C.CGContextAddLineToPoint(p.ctx, C.double(points[i]), C.double(points[i+1]))
		}
	}
	p.draw()
}
func (p *quartz) path(data string) error {
	tokens := pathTokens.FindAllString(data, -1)
	residue := pathTokens.ReplaceAllString(data, "")
	if strings.TrimSpace(strings.ReplaceAll(residue, ",", "")) != "" {
		return fmt.Errorf("unsupported generated path syntax")
	}
	C.CGContextBeginPath(p.ctx)
	x, y := 0., 0.
	for i := 0; i < len(tokens); {
		command := tokens[i]
		i++
		count := map[string]int{"M": 2, "L": 2, "H": 1, "V": 1, "Q": 4, "Z": 0}[command]
		if command != "M" && command != "L" && command != "H" && command != "V" && command != "Q" && command != "Z" {
			return fmt.Errorf("unsupported generated path command")
		}
		if i+count > len(tokens) {
			return fmt.Errorf("incomplete generated path")
		}
		v := make([]float64, count)
		for k := range count {
			var err error
			v[k], err = strconv.ParseFloat(tokens[i+k], 64)
			if err != nil {
				return err
			}
		}
		i += count
		switch command {
		case "M":
			x, y = v[0], v[1]
			C.CGContextMoveToPoint(p.ctx, C.double(x), C.double(y))
		case "L":
			x, y = v[0], v[1]
			C.CGContextAddLineToPoint(p.ctx, C.double(x), C.double(y))
		case "H":
			x = v[0]
			C.CGContextAddLineToPoint(p.ctx, C.double(x), C.double(y))
		case "V":
			y = v[0]
			C.CGContextAddLineToPoint(p.ctx, C.double(x), C.double(y))
		case "Q":
			x, y = v[2], v[3]
			C.CGContextAddQuadCurveToPoint(p.ctx, C.double(v[0]), C.double(v[1]), C.double(x), C.double(y))
		case "Z":
			C.CGContextClosePath(p.ctx)
		}
	}
	p.draw()
	return nil
}
func (p *quartz) text(x, y, size float64, text string) error {
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	if C.elab_pdf_text(p.ctx, value, C.double(size), C.double(x), C.double(y)) == 0 {
		return fmt.Errorf("cannot encode PDF label")
	}
	return nil
}
func (p *quartz) finish() ([]byte, error) {
	C.CGPDFContextEndPage(p.ctx)
	C.CGPDFContextClose(p.ctx)
	p.finished = true
	length := C.CFDataGetLength(C.CFDataRef(p.data))
	if length <= 0 || length > 8*1024*1024 {
		return nil, fmt.Errorf("PDF exceeds export bounds")
	}
	return C.GoBytes(unsafe.Pointer(C.CFDataGetBytePtr(C.CFDataRef(p.data))), C.int(length)), nil
}
func (p *quartz) close() {
	if !p.finished {
		C.CGPDFContextEndPage(p.ctx)
		C.CGPDFContextClose(p.ctx)
	}
	C.CGContextRelease(p.ctx)
	C.CFRelease(C.CFTypeRef(p.data))
}
