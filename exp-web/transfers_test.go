package main

import (
	"bytes"
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func TestBrowserAudioUsesCapturedWAVWithoutNativePlayback(t *testing.T) {
	f := fixture(t)
	f.command("example.load", workbench.ExampleLoadParams{ID: "555-tone", DiscardChanges: true})
	f.command("simulation.pause", struct{}{})
	f.wait()
	f.command("audio.render", workbench.AudioRenderParams{ID: "BZ1", Duration: .02, Gain: .1})
	deadline := time.Now().Add(10 * time.Second)
	var audio *workbench.AudioPreview
	for time.Now().Before(deadline) {
		audio = f.state().Audio
		if audio != nil && audio.Status != "running" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if audio == nil || audio.Status != "completed" || audio.Playing {
		t.Fatal("silent audio render failed or autoplayed", audio)
	}
	response := f.command("audio.read", workbench.IDParams{ID: audio.ID})
	raw, _ := json.Marshal(response.Result)
	var data workbench.AudioData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data.WAV, []byte("RIFF")) || len(data.WAV) != 524 {
		t.Fatal("invalid WAV audio", len(data.WAV))
	}
	download := f.request("POST", "/api/download?"+f.query("audio-wav"), nil, "")
	if download.Code != 200 || !bytes.Equal(download.Body.Bytes(), data.WAV) {
		t.Fatal("WAV export changed rendered samples", download.Body.String())
	}
	f.command("audio.stop", workbench.IDParams{ID: audio.ID})
}

func TestBrowserMacPDFAndPNGDownloads(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("native CoreGraphics export requires macOS; browser print remains portable")
	}
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	for _, kind := range []string{"pdf", "png"} {
		response := f.request("POST", "/api/download?"+f.query(kind), nil, "")
		signature := []byte("%PDF-")
		if kind == "png" {
			signature = []byte{137, 80, 78, 71, 13, 10, 26, 10}
		}
		if response.Code != 200 || !bytes.HasPrefix(response.Body.Bytes(), signature) {
			t.Fatalf("%s export: %d %s", kind, response.Code, response.Body.String())
		}
	}
}
