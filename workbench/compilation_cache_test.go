package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sync"
	"testing"
)

func requireSameCompilation(t *testing.T, cache *compilationCache, graph Graph) {
	t.Helper()
	want, status, diagnostic := graph.Compile()
	got, actualStatus, actualDiagnostic := cache.compile(graph)
	if !reflect.DeepEqual(want, got) || status != actualStatus || !reflect.DeepEqual(diagnostic, actualDiagnostic) {
		t.Fatalf("cached compilation differs from fresh compiler: status %q/%q diagnostic %+v/%+v", actualStatus, status, actualDiagnostic, diagnostic)
	}
	if !reflect.DeepEqual(graph.Summary(), graph.compiledSummary(got, actualDiagnostic)) {
		t.Fatal("summary changed")
	}
}

// Damage only mutable references, leaving ordinary copied scalar fields alone.
// Covers nested model pointers, block slices, nets/labels and terminal maps.
func damageCompilationReferences(value reflect.Value, borrowed bool) {
	switch value.Kind() {
	case reflect.Pointer:
		if !value.IsNil() {
			damageCompilationReferences(value.Elem(), true)
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			damageCompilationReferences(value.Field(i), borrowed)
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			damageCompilationReferences(value.Index(i), true)
		}
	case reflect.Map:
		if !value.IsNil() {
			value.Clear()
		}
	case reflect.String:
		if borrowed && value.CanSet() {
			value.SetString("damaged")
		}
	case reflect.Float64:
		if borrowed && value.CanSet() {
			value.SetFloat(12345)
		}
	case reflect.Int:
		if borrowed && value.CanSet() {
			value.SetInt(12345)
		}
	case reflect.Bool:
		if borrowed && value.CanSet() {
			value.SetBool(!value.Bool())
		}
	}
}

func TestCompilationCacheCatalogEqualityAndOwnership(t *testing.T) {
	for _, info := range ExampleCatalog() {
		t.Run(info.ID, func(t *testing.T) {
			def, _ := findExample(info.ID)
			graph := def.graph.Clone()
			before, err := json.Marshal(graph)
			if err != nil {
				t.Fatal(err)
			}
			cache := compilationCache{}
			requireSameCompilation(t, &cache, graph)
			requireSameCompilation(t, &cache, graph) // hit
			result, _, diagnostic := cache.compile(graph)
			damageCompilationReferences(reflect.ValueOf(&result).Elem(), false)
			if diagnostic != nil {
				diagnostic.Message = "damaged"
			}
			after, err := json.Marshal(graph)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatal("result borrowed source graph")
			}
			requireSameCompilation(t, &cache, graph)
			// Source ownership is independent too: changing the caller's graph cannot
			// rewrite the retained entry or an earlier returned compilation.
			retained := cloneCompiled(cache.compiled)
			damageCompilationReferences(reflect.ValueOf(&graph).Elem(), false)
			if !reflect.DeepEqual(retained, cache.compiled) {
				t.Fatal("entry borrowed source graph")
			}
		})
	}
}

func TestCompilationCacheMetadataAndInvalidation(t *testing.T) {
	fixtures := []Graph{{}, EmptyGraph(), ExampleGraph(), RCBlockExampleGraph(), capacityChain(t, "dc-source", 50), resistorCapacityChain(t, 63)}
	invalid := ExampleGraph()
	invalid.Wires[0].To = "missing.pin"
	fixtures = append(fixtures, invalid)
	cache := compilationCache{}
	for _, graph := range fixtures {
		requireSameCompilation(t, &cache, graph)
		requireSameCompilation(t, &cache, graph)
	}
	for _, info := range ExampleCatalog() {
		def, _ := findExample(info.ID)
		g := def.graph.Clone()
		requireSameCompilation(t, &cache, g)
		previous := cache.key
		g.Components[0].Position.X += 10
		requireSameCompilation(t, &cache, g)
		if cache.key == previous {
			t.Fatal("position not in identity", info.ID)
		}
		g.Components[0].Value += .01
		requireSameCompilation(t, &cache, g)
	}
	block := RCBlockExampleGraph()
	requireSameCompilation(t, &cache, block)
	previous := cache.key
	for i := range block.Components {
		if block.Components[i].Block != nil {
			block.Components[i].Block.Expanded = !block.Components[i].Block.Expanded
		}
	}
	requireSameCompilation(t, &cache, block)
	if previous == cache.key {
		t.Fatal("nested block metadata not in identity")
	}
	zero := ExampleGraph()
	zero.Components[0].Position.X = 0
	requireSameCompilation(t, &cache, zero)
	previous = cache.key
	zero.Components[0].Position.X = math.Copysign(0, -1)
	requireSameCompilation(t, &cache, zero)
	if previous == cache.key {
		t.Fatal("signed zero lost")
	}
	// Encoding errors must bypass, rather than poison or reuse, the last entry.
	original := cache.key
	bad := ExampleGraph()
	bad.Components[0].Value = math.NaN()
	_, wantStatus, wantDiagnostic := bad.Compile()
	_, status, diagnostic := cache.compile(bad)
	if status != wantStatus || !reflect.DeepEqual(diagnostic, wantDiagnostic) || cache.key != original {
		t.Fatal("nonfinite cache admission")
	}
}

func TestCompilationCacheSharedServiceConcurrentQueriesAndEdits(t *testing.T) {
	s := NewExample()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	var readers sync.WaitGroup
	errors := make(chan error, 4)
	for i := 0; i < 4; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for n := 0; n < 20; n++ {
				snap := s.Snapshot()
				snap.Components[0].ID = "borrowed"
				response := s.Dispatch(t.Context(), Request{Version: 1, ID: "reader", Command: "analysis.list", Params: json.RawMessage(`{}`)}, "agent")
				if response.Error != nil {
					errors <- fmt.Errorf("query: %v", response.Error)
					return
				}
			}
		}()
	}
	for n := 0; n < 10; n++ {
		call(t, s, "component.set", ValueParams{ID: "R1", Value: fmt.Sprintf("%d", 1000+n)}, fmt.Sprint("edit-", n))
	}
	readers.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	requireSameCompilation(t, &s.compilation, s.graph)
}

func BenchmarkCompilationBoundary(b *testing.B) {
	for _, id := range []string{"led", "discrete-class-ab", "rc-block"} {
		def, _ := findExample(id)
		graph := def.graph.Clone()
		b.Run(id+"/fresh", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				graph.Compile()
			}
		})
		b.Run(id+"/cached-owned", func(b *testing.B) {
			cache := compilationCache{}
			cache.compile(graph)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				cache.compile(graph)
			}
		})
	}
}

// Keep cache identity reviewable when graph/model structs evolve: every leaf
// must be exported JSON data, and changing it must change the serialized key.
func TestCompilationCacheEncodingCoversEveryGraphField(t *testing.T) {
	marshaler := reflect.TypeFor[json.Marshaler]()
	var populated func(reflect.Type) reflect.Value
	building := map[reflect.Type]bool{}
	populated = func(kind reflect.Type) reflect.Value {
		if building[kind] {
			t.Fatalf("recursive graph type requires explicit cache identity coverage: %s", kind)
		}
		building[kind] = true
		defer delete(building, kind)
		if kind.Implements(marshaler) || reflect.PointerTo(kind).Implements(marshaler) {
			t.Fatalf("custom graph encoding needs explicit cache coverage: %s", kind)
		}
		v := reflect.New(kind).Elem()
		switch kind.Kind() {
		case reflect.Pointer:
			v.Set(reflect.New(kind.Elem()))
			v.Elem().Set(populated(kind.Elem()))
		case reflect.Struct:
			for i := 0; i < kind.NumField(); i++ {
				f := kind.Field(i)
				if !f.IsExported() || f.Tag.Get("json") == "-" {
					t.Fatalf("field outside cache encoding: %s.%s", kind, f.Name)
				}
				v.Field(i).Set(populated(f.Type))
			}
		case reflect.Slice:
			v.Set(reflect.MakeSlice(kind, 1, 1))
			v.Index(0).Set(populated(kind.Elem()))
		case reflect.String:
			v.SetString("first")
		case reflect.Bool:
			v.SetBool(true)
		case reflect.Int:
			v.SetInt(1)
		case reflect.Float64:
			v.SetFloat(1)
		default:
			t.Fatalf("new graph kind requires cache identity coverage: %s", kind)
		}
		return v
	}
	graph := populated(reflect.TypeFor[Graph]())
	encoded := func() string {
		data, err := json.Marshal(graph.Interface())
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	baseline := encoded()
	var visit func(reflect.Value, string)
	visit = func(v reflect.Value, path string) {
		switch v.Kind() {
		case reflect.Pointer:
			visit(v.Elem(), path+"*")
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				visit(v.Field(i), path+"."+v.Type().Field(i).Name)
			}
		case reflect.Slice:
			visit(v.Index(0), path+"[0]")
		default:
			old := reflect.New(v.Type()).Elem()
			old.Set(v)
			switch v.Kind() {
			case reflect.String:
				v.SetString("second")
			case reflect.Bool:
				v.SetBool(false)
			case reflect.Int:
				v.SetInt(2)
			case reflect.Float64:
				v.SetFloat(2)
			}
			if encoded() == baseline {
				t.Error("graph leaf omitted from cache identity:", path)
			}
			v.Set(old)
		}
	}
	visit(graph, "Graph")
}
