package workbench

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCaptureProjectIsDetachedNativeData(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.rc", ReplaceParams{}, "capture-example")
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "capture-edit")
	before := s.Snapshot()
	capture, err := s.CaptureProject()
	if err != nil {
		t.Fatal(err)
	}
	if capture.CommandRevision != before.CommandRevision || capture.Generation != before.Document.Generation || !capture.Dirty {
		t.Fatal("capture identity differs from source")
	}
	document, err := decodeDocument(capture.Data)
	if err != nil {
		t.Fatal("not a valid native file", err)
	}
	if !reflect.DeepEqual(document.Graph.Components, before.Components) {
		t.Fatal("capture graph differs")
	}
	after := s.Snapshot()
	if after.CommandRevision != before.CommandRevision || after.Document.Dirty != before.Document.Dirty || after.Document.Path != before.Document.Path {
		t.Fatal("capture mutated save point")
	}
	capture.Data[0] = '!'
	again, err := s.CaptureProject()
	if err != nil || !json.Valid(again.Data) {
		t.Fatal("caller changed retained source data")
	}
}
