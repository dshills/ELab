package workbench

import (
	"fmt"

	"github.com/dshills/elab/internal/schematicpdf"
)

// SchematicPage is export/job configuration, not a saved circuit preference.
type SchematicPage struct {
	Paper        string   `json:"paper,omitempty"`
	Orientation  string   `json:"orientation,omitempty"`
	MarginMM     *float64 `json:"margin_mm,omitempty"`
	Scale        *float64 `json:"scale,omitempty"`
	Color        bool     `json:"color,omitempty"`
	Measurements bool     `json:"measurements,omitempty"`
}
type PDFExportParams struct {
	Path               string         `json:"path"`
	Overwrite          bool           `json:"overwrite,omitempty"`
	ExpectedGeneration *uint64        `json:"expected_document_generation,omitempty"`
	Page               *SchematicPage `json:"page,omitempty"`
}
type PNGExportParams struct {
	Path               string         `json:"path"`
	Overwrite          bool           `json:"overwrite,omitempty"`
	ExpectedGeneration *uint64        `json:"expected_document_generation,omitempty"`
	Page               *SchematicPage `json:"page,omitempty"`
	DPI                int            `json:"dpi,omitempty"` // 0 defaults to 150; otherwise 72–300
}

func cloneSchematicPage(in SchematicPage) SchematicPage {
	if in.MarginMM != nil {
		v := *in.MarginMM
		in.MarginMM = &v
	}
	if in.Scale != nil {
		v := *in.Scale
		in.Scale = &v
	}
	return in
}

func normalizeSchematicPage(in *SchematicPage) (SchematicPage, schematicpdf.Page, error) {
	page := SchematicPage{Paper: "letter", Orientation: "landscape"}
	margin := 10.
	if in != nil {
		page = *in
		if page.Paper == "" {
			page.Paper = "letter"
		}
		if page.Orientation == "" {
			page.Orientation = "landscape"
		}
		if page.MarginMM != nil {
			margin = *page.MarginMM
		}
	}
	width, height := 612., 792.
	switch page.Paper {
	case "letter":
	case "a4":
		width, height = 210*72/25.4, 297*72/25.4
	default:
		return page, schematicpdf.Page{}, fmt.Errorf("paper must be letter or a4")
	}
	switch page.Orientation {
	case "portrait":
	case "landscape":
		width, height = height, width
	default:
		return page, schematicpdf.Page{}, fmt.Errorf("orientation must be portrait or landscape")
	}
	if !finite(margin) || margin < 0 || margin > 40 {
		return page, schematicpdf.Page{}, fmt.Errorf("margin must be 0–40 mm")
	}
	scale := 0.
	if page.Scale != nil {
		scale = *page.Scale
		if !finite(scale) || scale < .1 || scale > 4 {
			return page, schematicpdf.Page{}, fmt.Errorf("scale must be 0.1–4, or omitted for fit")
		}
	}
	page.MarginMM = &margin
	if page.Scale != nil {
		v := scale
		page.Scale = &v
	}
	return page, schematicpdf.Page{Width: width, Height: height, Margin: margin * 72 / 25.4, Scale: scale, Grayscale: !page.Color}, nil
}
