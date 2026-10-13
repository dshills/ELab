package workbench

import (
	"context"
	"time"
)

const RecordingPlotBuckets = 128

type RecordingViewParams struct {
	Mode string `json:"mode"`
}
type RecordingSeekParams struct {
	Index *int     `json:"index,omitempty"`
	Time  *float64 `json:"time_s,omitempty"`
}
type RecordingStepParams struct {
	Direction string `json:"direction"`
}
type RecordingPlayParams struct {
	Speed float64 `json:"seconds_per_wall_second"`
}
type RecordingWindowParams struct {
	Start float64 `json:"start_s"`
	End   float64 `json:"end_s"`
}
type RecordingView struct {
	ID            string                `json:"id"`
	OriginID      string                `json:"origin_id"`
	Imported      bool                  `json:"imported"`
	Viewing       bool                  `json:"viewing"`
	Playing       bool                  `json:"playing"`
	Speed         float64               `json:"seconds_per_wall_second"`
	Index         int                   `json:"sample_index"`
	Time          float64               `json:"time_s"`
	Duration      float64               `json:"duration_s"`
	Step          float64               `json:"sample_step_s"`
	Total         int                   `json:"sample_count"`
	WindowSamples int                   `json:"window_sample_count"`
	PlotReduced   bool                  `json:"plot_reduced"`
	Scope         Scope                 `json:"scope"`
	Readings      []VerificationReading `json:"readings"`
}

type recordingState struct {
	viewing, playing bool
	index            int
	speed            float64
	window           *RecordingWindowParams
	probes           []string
	a, b             *float64
	activeCursor     string
	cancel           context.CancelFunc
	wall             time.Time
	anchor           float64
}
