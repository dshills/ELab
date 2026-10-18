package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dshills/vellum/drivers/cogent"
)

type nativeFocusObservation struct {
	Requests, Admissions int
	FocusedID            string
	Generation           uint64
}

func waitNativeAccessibilityFocus(ctx context.Context, domainID string,
	inspect func(context.Context) (cogent.AccessibilityInspection, error),
	request func(context.Context, string) (bool, error),
) (nativeFocusObservation, error) {
	observed := nativeFocusObservation{}
	failure := func(err error) (nativeFocusObservation, error) {
		return observed, fmt.Errorf("native focus %q not observed after %d requests/%d admissions; focused=%q generation=%d: %w", domainID, observed.Requests, observed.Admissions, observed.FocusedID, observed.Generation, err)
	}
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return failure(err)
		}
		inspection, err := inspect(ctx)
		if err != nil {
			return failure(err)
		}
		observed.FocusedID, observed.Generation = inspection.FocusedID, inspection.Generation
		if err := ctx.Err(); err != nil {
			return failure(err)
		}
		target := ""
		for _, element := range inspection.Elements {
			if inspection.Reachable && inspection.NativeWindow && element.DomainID == domainID && element.Enabled &&
				!element.GridPending && element.ID != "" && element.Frame[2] > 0 && element.Frame[3] > 0 && element.HitTestID == element.ID {
				target = element.ID
				if target == inspection.FocusedID {
					return observed, nil
				}
			}
		}
		if err := ctx.Err(); err != nil {
			return failure(err)
		}
		if target != "" {
			// Admission can be retired by a semantic publication before owner
			// delivery. Retry only this idempotent focus request against a fresh
			// observation; editing and Apply must never be repeated here.
			observed.Requests++
			accepted, err := request(ctx, target)
			if err != nil {
				return failure(err)
			}
			if accepted {
				observed.Admissions++
			}
		}
		select {
		case <-ctx.Done():
			return failure(ctx.Err())
		case <-tick.C:
		}
	}
}
