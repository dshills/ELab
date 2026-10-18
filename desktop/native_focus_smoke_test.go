package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dshills/vellum/drivers/cogent"
)

func focusInspection(id, focused string) cogent.AccessibilityInspection {
	return cogent.AccessibilityInspection{Generation: 17, Reachable: true, NativeWindow: true, FocusedID: focused,
		Elements: []cogent.AccessibilityElement{{ID: id, DomainID: "value-entry", Enabled: true, Frame: [4]float64{10, 10, 100, 30}, HitTestID: id}}}
}

func TestNativeFocusWaitObservesCompletionAfterLostAdmission(t *testing.T) {
	for _, firstAdmitted := range []bool{false, true} {
		name := "refused-before-queue"
		if firstAdmitted {
			name = "retired-after-admission"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			calls, inspections := 0, 0
			focused := "wire-selector"
			observed, err := waitNativeAccessibilityFocus(ctx, "value-entry", func(context.Context) (cogent.AccessibilityInspection, error) {
				inspections++
				id := "current-value-field"
				if calls == 0 {
					id = "old-value-field"
				}
				return focusInspection(id, focused), nil
			}, func(_ context.Context, id string) (bool, error) {
				calls++
				if calls == 1 {
					return firstAdmitted, nil // no completion: generation changes before delivery
				}
				if id != "current-value-field" {
					t.Fatal("retry used an obsolete target", id)
				}
				focused = id
				return true, nil
			})
			if err != nil || calls != 2 || inspections < 3 || observed.FocusedID != "current-value-field" {
				t.Fatal("admission was mistaken for completion or no fresh retry occurred", calls, inspections, observed, err)
			}
			wantAdmissions := 1
			if firstAdmitted {
				wantAdmissions++
			}
			if observed.Requests != 2 || observed.Admissions != wantAdmissions {
				t.Fatal("focus evidence lost actual request/admission counts", observed)
			}
		})
	}
}

func TestNativeFocusWaitNeverTreatsAdmissionAsSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 65*time.Millisecond)
	defer cancel()
	observed, err := waitNativeAccessibilityFocus(ctx, "value-entry", func(context.Context) (cogent.AccessibilityInspection, error) {
		return focusInspection("field", "wire-selector"), nil
	}, func(context.Context, string) (bool, error) { return true, nil })
	if !errors.Is(err, context.DeadlineExceeded) || observed.FocusedID != "wire-selector" {
		t.Fatal("an admitted focus with no observed completion passed", observed, err)
	}
}

func TestNativeFocusWaitRejectsObservationAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := waitNativeAccessibilityFocus(ctx, "value-entry", func(context.Context) (cogent.AccessibilityInspection, error) {
		cancel()
		return focusInspection("field", "field"), nil
	}, func(context.Context, string) (bool, error) {
		t.Fatal("canceled waiter requested focus")
		return false, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("observation after cancellation passed", err)
	}
}

func TestNativeFocusWaitUsesCurrentNativeTarget(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	requests := 0
	observed, err := waitNativeAccessibilityFocus(ctx, "value-entry", func(context.Context) (cogent.AccessibilityInspection, error) {
		// A focused retired ID must not qualify its same-domain replacement.
		focused := "retired-value-field"
		if requests > 0 {
			focused = "current-value-field"
		}
		return focusInspection("current-value-field", focused), nil
	}, func(_ context.Context, id string) (bool, error) {
		requests++
		if id != "current-value-field" {
			t.Fatal("focus request used an old native identifier", id)
		}
		return true, nil
	})
	if err != nil || requests != 1 || observed.FocusedID != "current-value-field" {
		t.Fatal("replacement focus was not actually observed", observed, err)
	}
	_, err = waitNativeAccessibilityFocus(ctx, "value-entry", func(context.Context) (cogent.AccessibilityInspection, error) {
		return focusInspection("current-value-field", "current-value-field"), nil
	}, func(context.Context, string) (bool, error) {
		t.Fatal("already focused control was requested again")
		return false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNativeFocusWaitDoesNotActOnUnavailableTarget(t *testing.T) {
	for _, kind := range []string{"missing", "disabled", "pending", "zero-frame", "obscured", "unreachable", "not-native"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			state := focusInspection("field", "field")
			switch kind {
			case "missing":
				state.Elements = nil
			case "disabled":
				state.Elements[0].Enabled = false
			case "pending":
				state.Elements[0].GridPending = true
			case "zero-frame":
				state.Elements[0].Frame = [4]float64{}
			case "obscured":
				state.Elements[0].HitTestID = "overlay"
			case "unreachable":
				state.Reachable = false
			case "not-native":
				state.NativeWindow = false
			}
			_, err := waitNativeAccessibilityFocus(ctx, "value-entry", func(context.Context) (cogent.AccessibilityInspection, error) {
				cancel()
				return state, nil
			}, func(context.Context, string) (bool, error) {
				t.Fatal("unavailable control received focus")
				return false, nil
			})
			if !errors.Is(err, context.Canceled) {
				t.Fatal("unavailable control qualified focus", err)
			}
		})
	}
}

func TestNativeFocusWaitPreservesCancellationAndNativeErrors(t *testing.T) {
	sentinel := errors.New("native request failed")
	for _, stage := range []string{"pre-canceled", "inspect", "request"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			want := sentinel
			if stage == "pre-canceled" {
				cancel()
				want = context.Canceled
			}
			_, err := waitNativeAccessibilityFocus(ctx, "value-entry", func(context.Context) (cogent.AccessibilityInspection, error) {
				if stage == "pre-canceled" {
					t.Fatal("canceled waiter queried native controls")
				}
				if stage == "inspect" {
					return cogent.AccessibilityInspection{}, sentinel
				}
				return focusInspection("field", "wire-selector"), nil
			}, func(context.Context, string) (bool, error) {
				if stage != "request" {
					t.Fatal("request continued after cancellation or native query error")
				}
				return false, sentinel
			})
			if !errors.Is(err, want) {
				t.Fatal("lost native/context error", err)
			}
		})
	}
}
