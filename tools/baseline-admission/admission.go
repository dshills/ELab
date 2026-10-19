package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"strings"
)

const (
	maxProfileBytes = 64 << 10
	maxPNGBytes     = 64 << 20
	maxImagePixels  = 4 << 20
	maxSamplePixels = 1 << 20
)

type region struct {
	Name      string `json:"name"`
	Rect      []int  `json:"rect"`
	RGBSHA256 string `json:"rgb_sha256"`
}

type profile struct {
	Version int      `json:"version"`
	Width   int      `json:"width"`
	Height  int      `json:"height"`
	Regions []region `json:"regions"`
}

type regionResult struct {
	Name      string `json:"name"`
	Matches   bool   `json:"matches"`
	RGBSHA256 string `json:"rgb_sha256,omitempty"`
}

type admission struct {
	Version       int            `json:"version"`
	Admitted      bool           `json:"admitted"`
	Code          string         `json:"code"`
	Message       string         `json:"message,omitempty"`
	ProfileSHA256 string         `json:"profile_sha256,omitempty"`
	ImageSHA256   string         `json:"image_sha256,omitempty"`
	Width         int            `json:"width,omitempty"`
	Height        int            `json:"height,omitempty"`
	Regions       []regionResult `json:"regions,omitempty"`
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func validDigest(value string) bool {
	raw, err := hex.DecodeString(value)
	return err == nil && len(raw) == sha256.Size
}

func readBounded(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("expected a regular file of at most %d bytes", limit)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	value, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(value)) > limit {
		return nil, fmt.Errorf("file exceeds %d bytes", limit)
	}
	return value, nil
}

func decodeProfile(value []byte) (profile, error) {
	var p profile
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return p, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return p, fmt.Errorf("profile must contain one JSON object")
	}
	if p.Version != 1 || p.Width < 1 || p.Height < 1 || p.Width > 4096 || p.Height > 4096 || p.Width*p.Height > maxImagePixels {
		return p, fmt.Errorf("unsupported profile version or dimensions")
	}
	if len(p.Regions) < 1 || len(p.Regions) > 16 {
		return p, fmt.Errorf("profile needs 1–16 regions")
	}
	names := make(map[string]bool)
	pixels := 0
	for _, r := range p.Regions {
		if r.Name == "" || len(r.Name) > 64 || strings.TrimSpace(r.Name) != r.Name || names[r.Name] || !validDigest(r.RGBSHA256) || len(r.Rect) != 4 {
			return p, fmt.Errorf("invalid or duplicate region")
		}
		names[r.Name] = true
		x0, y0, x1, y1 := r.Rect[0], r.Rect[1], r.Rect[2], r.Rect[3]
		if x0 < 0 || y0 < 0 || x1 > p.Width || y1 > p.Height || x0 >= x1 || y0 >= y1 {
			return p, fmt.Errorf("region %s is outside the image", r.Name)
		}
		pixels += (x1 - x0) * (y1 - y0)
		if pixels > maxSamplePixels {
			return p, fmt.Errorf("region pixel budget exceeded")
		}
	}
	return p, nil
}

// Admission checks an existing PNG only. It neither captures a window nor
// establishes its identity, focus or continued visibility. The recorder and
// complete-frame analysis retain those independent responsibilities.
func inspectBaseline(profileBytes, pngBytes []byte, expectedImageSHA string) admission {
	out := admission{Version: 1, ProfileSHA256: digest(profileBytes)}
	reject := func(code, message string) admission {
		out.Code, out.Message = code, message
		return out
	}
	if len(profileBytes) > maxProfileBytes {
		return reject("invalid-profile", "profile byte budget exceeded")
	}
	p, err := decodeProfile(profileBytes)
	if err != nil {
		return reject("invalid-profile", err.Error())
	}
	if !validDigest(expectedImageSHA) {
		return reject("invalid-image-sha256", "expected the recorder's complete PNG SHA-256")
	}
	if len(pngBytes) > maxPNGBytes {
		return reject("image-too-large", "PNG byte budget exceeded")
	}
	out.ImageSHA256 = digest(pngBytes)
	if out.ImageSHA256 != strings.ToLower(expectedImageSHA) {
		return reject("image-sha256-mismatch", "PNG differs from the recorder frame")
	}
	config, err := png.DecodeConfig(bytes.NewReader(pngBytes))
	if err != nil {
		return reject("invalid-png", err.Error())
	}
	out.Width, out.Height = config.Width, config.Height
	if config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 || config.Width*config.Height > maxImagePixels {
		return reject("image-too-large", "PNG decoded pixel budget exceeded")
	}
	if config.Width != p.Width || config.Height != p.Height {
		return reject("dimension-mismatch", "PNG dimensions differ from the declared profile")
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return reject("invalid-png", err.Error())
	}
	matched := true
	for _, r := range p.Regions {
		box := image.Rect(r.Rect[0], r.Rect[1], r.Rect[2], r.Rect[3])
		pixels := make([]byte, 0, box.Dx()*box.Dy()*3)
		for y := box.Min.Y; y < box.Max.Y; y++ {
			for x := box.Min.X; x < box.Max.X; x++ {
				c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				if c.A != 255 {
					return reject("nonopaque-region", "declared display regions must be opaque")
				}
				pixels = append(pixels, c.R, c.G, c.B)
			}
		}
		actual := digest(pixels)
		result := regionResult{Name: r.Name, Matches: actual == strings.ToLower(r.RGBSHA256), RGBSHA256: actual}
		out.Regions = append(out.Regions, result)
		matched = matched && result.Matches
	}
	if !matched {
		return reject("region-mismatch", "baseline does not match the declared ELab regions")
	}
	out.Admitted, out.Code = true, "admitted"
	return out
}
