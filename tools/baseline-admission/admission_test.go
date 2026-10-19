package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, change func(*image.NRGBA)) ([]byte, []byte) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{1, 2, 3, 255})
	img.SetNRGBA(1, 0, color.NRGBA{4, 5, 6, 255})
	img.SetNRGBA(0, 1, color.NRGBA{7, 8, 9, 255})
	img.SetNRGBA(1, 1, color.NRGBA{10, 11, 12, 255})
	if change != nil {
		change(img)
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	p := profile{Version: 1, Width: 2, Height: 2, Regions: []region{
		{Name: "known-row", Rect: []int{0, 0, 2, 1}, RGBSHA256: digest([]byte{1, 2, 3, 4, 5, 6})},
	}}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return raw, encoded.Bytes()
}

func TestBaselineRegionsAndEncodedIdentity(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*image.NRGBA)
		code   string
	}{
		{"known", nil, "admitted"},
		{"covered-region", func(img *image.NRGBA) { img.SetNRGBA(0, 0, color.NRGBA{99, 2, 3, 255}) }, "region-mismatch"},
		{"transparent-region", func(img *image.NRGBA) { img.SetNRGBA(0, 0, color.NRGBA{1, 2, 3, 128}) }, "nonopaque-region"},
		// Deliberately demonstrates the guard's limit: unobserved pixels do
		// not prove focus or continued visibility. Full-frame admission stays.
		{"outside-region", func(img *image.NRGBA) { img.SetNRGBA(1, 1, color.NRGBA{99, 99, 99, 255}) }, "admitted"},
	} {
		t.Run(test.name, func(t *testing.T) {
			p, img := fixture(t, test.change)
			result := inspectBaseline(p, img, digest(img))
			if result.Code != test.code || result.Admitted != (test.code == "admitted") {
				t.Fatalf("unexpected decision: %+v", result)
			}
			if result.ProfileSHA256 != digest(p) || result.ImageSHA256 != digest(img) {
				t.Fatal("decision lost its exact data identities")
			}
		})
	}
	p, img := fixture(t, nil)
	if got := inspectBaseline(p, img, strings.Repeat("0", 64)); got.Admitted || got.Code != "image-sha256-mismatch" {
		t.Fatal("image identity did not guard pixel matching", got)
	}
	if got := inspectBaseline(p, img, "not-a-digest"); got.Admitted || got.Code != "invalid-image-sha256" {
		t.Fatal("malformed frame identity admitted", got)
	}
}

func TestInvalidProfilesFailClosed(t *testing.T) {
	p, img := fixture(t, nil)
	for _, test := range []struct {
		name   string
		change func(*profile)
	}{
		{"version", func(p *profile) { p.Version = 2 }},
		{"zero-width", func(p *profile) { p.Width = 0 }},
		{"huge-width", func(p *profile) { p.Width = int(^uint(0) >> 1) }},
		{"empty-regions", func(p *profile) { p.Regions = nil }},
		{"duplicate-name", func(p *profile) { p.Regions = append(p.Regions, p.Regions[0]) }},
		{"bad-hash", func(p *profile) { p.Regions[0].RGBSHA256 = "bad" }},
		{"extra-coordinate", func(p *profile) { p.Regions[0].Rect = []int{0, 0, 2, 1, 0} }},
		{"missing-coordinate", func(p *profile) { p.Regions[0].Rect = []int{0, 0, 2} }},
		{"negative-coordinate", func(p *profile) { p.Regions[0].Rect = []int{-1, 0, 2, 1} }},
		{"outside-image", func(p *profile) { p.Regions[0].Rect = []int{0, 0, 3, 1} }},
		{"inverted-region", func(p *profile) { p.Regions[0].Rect = []int{2, 0, 1, 1} }},
		{"pixel-budget", func(p *profile) {
			p.Width, p.Height = 2048, 2048
			p.Regions[0].Rect = []int{0, 0, 2048, 2048}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var changed profile
			if err := json.Unmarshal(p, &changed); err != nil {
				t.Fatal(err)
			}
			test.change(&changed)
			raw, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			if got := inspectBaseline(raw, img, digest(img)); got.Admitted || got.Code != "invalid-profile" {
				t.Fatal("invalid profile admitted", got)
			}
		})
	}
	for _, raw := range [][]byte{
		append(append([]byte{}, p...), []byte(" {}")...),
		[]byte(strings.Replace(string(p), `"version":1`, `"version":1,"unknown":true`, 1)),
		[]byte("null"),
		bytes.Repeat([]byte(" "), maxProfileBytes+1),
	} {
		if got := inspectBaseline(raw, img, digest(img)); got.Admitted || got.Code != "invalid-profile" {
			t.Fatal("ambiguous or oversized profile admitted", got)
		}
	}
}

func TestPNGDimensionsAndIntegrityBeforeAdmission(t *testing.T) {
	p, img := fixture(t, nil)
	for _, test := range []struct {
		name string
		data []byte
		code string
	}{
		{"not-png", []byte("not an image"), "invalid-png"},
		{"truncated", append([]byte{}, img[:40]...), "invalid-png"},
		{"oversized-header", changeDimensions(img, 5000, 5000), "image-too-large"},
		{"wrong-dimensions", changeDimensions(img, 3, 2), "dimension-mismatch"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := inspectBaseline(p, test.data, digest(test.data))
			if got.Admitted || got.Code != test.code {
				t.Fatal("invalid image admitted", got)
			}
		})
	}
	corrupt := append([]byte{}, img...)
	corrupt[len(corrupt)-1] ^= 1
	if got := inspectBaseline(p, corrupt, digest(corrupt)); got.Admitted || got.Code != "invalid-png" {
		t.Fatal("bad PNG checksum admitted", got)
	}
}

func changeDimensions(pngBytes []byte, width, height uint32) []byte {
	changed := append([]byte{}, pngBytes...)
	binary.BigEndian.PutUint32(changed[16:20], width)
	binary.BigEndian.PutUint32(changed[20:24], height)
	binary.BigEndian.PutUint32(changed[29:33], crc32.ChecksumIEEE(changed[12:29]))
	return changed
}

func TestBoundedFilesAndCLI(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "bytes")
	if err := os.WriteFile(file, []byte("four"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readBounded(file, 4); err != nil {
		t.Fatal(err)
	}
	if _, err := readBounded(file, 3); err == nil {
		t.Fatal("oversized file admitted")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{link, dir, filepath.Join(dir, "missing")} {
		if _, err := readBounded(path, 32); err == nil {
			t.Fatal("non-regular input admitted", path)
		}
	}
	p, img := fixture(t, nil)
	profilePath, imagePath := filepath.Join(dir, "profile.json"), filepath.Join(dir, "frame.png")
	if err := os.WriteFile(profilePath, p, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imagePath, img, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		args []string
		code int
	}{
		{"admit", []string{"--profile", profilePath, "--image", imagePath, "--sha256", digest(img)}, 0},
		{"reject", []string{"--profile", profilePath, "--image", imagePath, "--sha256", strings.Repeat("0", 64)}, 2},
		{"missing-argument", nil, 1},
		{"unknown-argument", []string{"--unknown"}, 1},
		{"bad-profile-path", []string{"--profile", link, "--image", imagePath, "--sha256", digest(img)}, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(test.args, &stdout, &stderr)
			var result admission
			if code != test.code || json.Unmarshal(stdout.Bytes(), &result) != nil || result.Admitted != (code == 0) {
				t.Fatalf("wrong CLI decision: %d %s %s", code, stdout.String(), stderr.String())
			}
		})
	}
	if code := run([]string{"--profile", profilePath, "--image", imagePath, "--sha256", digest(img)}, failingWriter{}, &bytes.Buffer{}); code != 1 {
		t.Fatal("unwritten receipt reported success")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("injected output failure") }
