package main

import (
	"math"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func TestWeb555LiveClockSurvivesCPUContention(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	f := fixture(t)
	f.command("example.load", workbench.ExampleLoadParams{ID: "555-tone", DiscardChanges: true})
	f.command("simulation.pause", struct{}{})
	f.wait()
	stop, ready := make(chan struct{}), make(chan struct{}, 8)
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			ready <- struct{}{}
			value := 1.
			for {
				select {
				case <-stop:
					runtime.KeepAlive(value)
					return
				default:
					for j := 0; j < 2048; j++ {
						value = math.Sqrt(value + 1.2345)
					}
				}
			}
		}()
	}
	defer func() { close(stop); workers.Wait() }()
	for i := 0; i < 8; i++ {
		<-ready
	}
	start := f.state().Transient.Sample.Time
	f.command("simulation.resume", struct{}{})
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		state := f.state()
		if state.Status == "error" || state.Transient.Stale {
			t.Fatal("CPU contention stopped a valid live circuit", state.Diagnostics)
		}
		if state.Transient.Sample.Time >= start+.001 {
			f.command("simulation.pause", struct{}{})
			if state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
				t.Fatal("accepted live sample violates conservation")
			}
			t.Logf("live 555 advanced %.6g s under eight burners on one Go scheduler CPU", state.Transient.Sample.Time-start)
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("live clock failed to progress under CPU contention")
}
