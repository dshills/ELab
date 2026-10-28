package workbench

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/dshills/elab/internal/audioplay"
)

const audioRate = 12000
const maxAudioSamples = 3000
const speakerPreviewFloorVolts = 1e-6

// Preview uses an independent experiment, never the editor's live state.
type AudioRenderParams struct {
	ID       string  `json:"id"`
	Start    float64 `json:"start_s"`
	Duration float64 `json:"duration_s"`
	Gain     float64 `json:"gain"`
}
type AudioPreview struct {
	Signal             string   `json:"signal,omitempty"`
	ID                 string   `json:"id"`
	Component          string   `json:"component_id"`
	Status             string   `json:"status"`
	Playing            bool     `json:"playing"`
	Current            bool     `json:"source_current"`
	ElectricalRevision uint64   `json:"electrical_revision"`
	DocumentGeneration uint64   `json:"document_generation"`
	Start              float64  `json:"start_s"`
	Duration           float64  `json:"duration_s"`
	Rate               int      `json:"sample_rate_hz"`
	Gain               float64  `json:"gain"`
	Bytes              int      `json:"bytes"`
	SHA256             string   `json:"sha256,omitempty"`
	Error              *Failure `json:"error,omitempty"`
}
type AudioData struct {
	Preview AudioPreview `json:"preview"`
	WAV     []byte       `json:"wav_base64"`
}
type audioJob struct {
	view   AudioPreview
	ctx    context.Context
	cancel context.CancelFunc
	wav    []byte
	hash   [32]byte
}

func (s *Service) audioViewLocked() *AudioPreview {
	if s.audio == nil {
		return nil
	}
	v := s.audio.view
	v.Current = !s.closed && v.DocumentGeneration == s.documentGeneration && s.audio.hash == contentHash(s.graph, s.transientSettings)
	if v.Error != nil {
		e := *v.Error
		v.Error = &e
	}
	return &v
}
func (s *Service) audioPlayerLocked() audioplay.Player {
	if s.audioPlayer == nil {
		s.audioPlayer = audioplay.New()
	}
	return s.audioPlayer
}
func (s *Service) stopAudioClockLocked() {
	s.audioPlaybackSerial++
	if s.audioPlaybackCancel != nil {
		s.audioPlaybackCancel()
		s.audioPlaybackCancel = nil
	}
}
func (s *Service) stopAudioLocked() {
	s.stopAudioClockLocked()
	if s.audioPlayer != nil {
		s.audioPlayer.Stop()
	}
	if s.audio != nil {
		s.audio.view.Playing = false
	}
}
func (s *Service) cancelAudioLocked() {
	s.stopAudioLocked()
	if s.audio != nil {
		s.audio.cancel()
	}
}
func (s *Service) audioCommandLocked(command string, raw []byte) error {
	if command == "audio.render" {
		var p AudioRenderParams
		if err := params(raw, &p, "id", "start_s", "duration_s", "gain"); err != nil {
			return err
		}
		return s.startAudioLocked(p)
	}
	var p IDParams
	if err := params(raw, &p, "id"); err != nil {
		return err
	}
	if s.audio == nil || s.audio.view.ID != p.ID {
		return fmt.Errorf("choose the current audio preview ID")
	}
	if command == "audio.stop" {
		s.cancelAudioLocked()
		return nil
	}
	if command != "audio.play" {
		return fmt.Errorf("unsupported audio action")
	}
	view := s.audioViewLocked()
	if view.Status != "completed" || !view.Current {
		return fmt.Errorf("render a current completed audio preview before playing")
	}
	if err := s.audioPlayerLocked().Play(s.audio.wav); err != nil {
		return &actionError{"audio-unavailable", err.Error()}
	}
	s.stopAudioClockLocked()
	s.audio.view.Playing = true
	job := s.audio
	playctx, cancel := context.WithCancel(context.Background())
	s.audioPlaybackCancel = cancel
	serial := s.audioPlaybackSerial
	// Completion refers to scheduled playback, not a claim about speaker hardware.
	s.liveWorkers.Add(1)
	go func() {
		defer s.liveWorkers.Done()
		timer := time.NewTimer(time.Duration(view.Duration * float64(time.Second)))
		defer timer.Stop()
		select {
		case <-playctx.Done():
		case <-timer.C:
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.audio == job && serial == s.audioPlaybackSerial && job.view.Playing {
			s.stopAudioLocked()
			s.revision++
			s.signalLocked("preview")
		}
	}()
	return nil
}
func (s *Service) startAudioLocked(p AudioRenderParams) error {
	if !finite(p.Start) || p.Start < 0 || p.Start > 1 || !finite(p.Duration) || p.Duration < .02 || p.Duration > .25 || !finite(p.Gain) || p.Gain < 0 || p.Gain > .25 {
		return fmt.Errorf("audio start must be 0–1 s, duration 0.02–0.25 s and gain 0–0.25")
	}
	// Publish the actual sample grid used by the WAV, rather than sub-sample
	// requested times that cannot be represented at the preview rate.
	p.Start = math.Round(p.Start*audioRate) / audioRate
	p.Duration = math.Round(p.Duration*audioRate) / audioRate
	if s.audioWorkers != 0 {
		return &actionError{"busy", "Finish or stop the current audio render first"}
	}
	part, ok := s.graph.Component(p.ID)
	if !ok || !isSoundKind(part.Kind) && part.Kind != "speaker" {
		return fmt.Errorf("choose a loudspeaker, piezo or active buzzer")
	}
	c, status, d := s.compileGraphLocked()
	if status != "" || d != nil {
		return fmt.Errorf("complete a supported circuit before rendering audio")
	}
	s.cancelAudioLocked()
	s.audioSerial++
	ctx, cancel := context.WithCancel(context.Background())
	job := &audioJob{ctx: ctx, cancel: cancel, hash: contentHash(s.graph, s.transientSettings), view: AudioPreview{ID: fmt.Sprintf("audio-%d", s.audioSerial), Component: p.ID, Status: "running", ElectricalRevision: s.electrical, DocumentGeneration: s.documentGeneration, Start: p.Start, Duration: p.Duration, Rate: audioRate, Gain: p.Gain}}
	if part.Kind == "speaker" {
		job.view.Signal = "Terminal voltage; 100 Hz high-pass; peak normalized above 1 µV before selected gain; no calibrated acoustic output"
	}
	s.audio = job
	s.audioWorkers++
	s.liveWorkers.Add(1)
	settings := s.transientSettings
	go s.runAudio(job, c, settings, part, p)
	return nil
}
func (s *Service) runAudio(job *audioJob, c Compiled, settings TransientSettings, part Component, p AudioRenderParams) {
	defer s.liveWorkers.Done()
	ctx, cancel := context.WithTimeout(job.ctx, 30*time.Second)
	defer cancel()
	data, err := renderAudio(ctx, c, settings, part, p)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audioWorkers--
	if s.audio != job || s.closed {
		return
	}
	if job.ctx.Err() != nil {
		err = job.ctx.Err()
	}
	if err != nil {
		job.view.Status = "failed"
		if job.ctx.Err() != nil {
			job.view.Status = "canceled"
		}
		job.view.Error = &Failure{"audio-render-failed", err.Error()}
	} else {
		job.wav = data
		job.view.Bytes = len(data)
		digest := sha256.Sum256(data)
		job.view.SHA256 = hex.EncodeToString(digest[:])
		job.view.Status = "completed"
	}
	s.revision++
	s.signalLocked("preview", "jobs")
}
func renderAudio(ctx context.Context, c Compiled, settings TransientSettings, part Component, p AudioRenderParams) ([]byte, error) {
	settings.StepSeconds = 1.0 / audioRate
	settings.PlaybackSpeed = 1
	kernel, err := newSimulationKernel(ctx, kernelPlan{compiled: c, settings: settings, mode: "auto", run: 1, revision: 1})
	if err != nil {
		return nil, err
	}
	sample := func() (float64, error) {
		if kernel.mixed != nil {
			for _, r := range kernel.mixed.result.Speakers {
				if r.ID == part.ID {
					return r.Voltage, nil
				}
			}
			for _, r := range kernel.mixed.result.Sounds {
				if r.ID == part.ID {
					return r.Voltage, nil
				}
			}
		}
		return 0, fmt.Errorf("sounder not resolved")
	}
	count := int(math.Round(p.Duration * audioRate))
	if count > maxAudioSamples {
		return nil, fmt.Errorf("audio sample budget exceeded")
	}
	pre := int(math.Round(p.Start * audioRate))
	pcm := make([]int16, count)
	var speakerSamples []float64
	if part.Kind == "speaker" {
		speakerSamples = make([]float64, count)
	}
	previous, y := 0., 0.
	alpha := math.Exp(-2 * math.Pi * 100 / audioRate)
	for k := 0; k < pre+count; k++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if k > 0 {
			if err := kernel.advance(ctx, float64(k)/audioRate); err != nil {
				return nil, err
			}
		}
		v, err := sample()
		if err != nil {
			return nil, err
		}
		a := 0.
		if k == 0 && part.Kind == "speaker" {
			previous = v
		}
		if part.Kind == "piezo" || part.Kind == "speaker" {
			y = alpha * (y + v - previous)
			previous = v
			a = y / 5
			if part.Kind == "speaker" {
				a = y
			}
		} else {
			en, _ := smoothLevel(v, 2, 3)
			a = en * math.Sin(2*math.Pi*part.Device.Sound.Frequency*float64(k)/audioRate)
		}
		if k >= pre {
			index := k - pre
			if speakerSamples != nil {
				if !finite(a) {
					return nil, fmt.Errorf("nonfinite speaker preview")
				}
				speakerSamples[index] = a
				continue
			}
			fade := audioPreviewFade(index, count)
			value := math.Max(-1, math.Min(1, a)) * p.Gain * fade
			pcm[index] = int16(math.Round(value * 32767))
		}
	}
	if speakerSamples != nil {
		fillSpeakerPCM(pcm, speakerSamples, p.Gain)
	}
	return encodeAudioWAV(pcm), nil
}
func audioPreviewFade(index, count int) float64 {
	return math.Min(1, math.Min(float64(index)/60, float64(count-1-index)/60))
}

// Only preview normalization uses this floor; electrical accuracy/work guards do not change.
func fillSpeakerPCM(pcm []int16, samples []float64, gain float64) {
	peak := 0.
	for _, v := range samples {
		peak = math.Max(peak, math.Abs(v))
	}
	if peak < speakerPreviewFloorVolts {
		return
	}
	for i, v := range samples {
		pcm[i] = int16(math.Round(v / peak * gain * audioPreviewFade(i, len(samples)) * 32767))
	}
}

func encodeAudioWAV(pcm []int16) []byte {
	b := new(bytes.Buffer)
	b.WriteString("RIFF")
	_ = binary.Write(b, binary.LittleEndian, uint32(36+2*len(pcm)))
	b.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(audioRate), uint32(audioRate * 2), uint16(2), uint16(16)} {
		_ = binary.Write(b, binary.LittleEndian, v)
	}
	b.WriteString("data")
	_ = binary.Write(b, binary.LittleEndian, uint32(len(pcm)*2))
	_ = binary.Write(b, binary.LittleEndian, pcm)
	return b.Bytes()
}
