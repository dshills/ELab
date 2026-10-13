package workbench

import (
	"encoding/json"
	"testing"
)

func TestDiscoveryRegistryResolvesSnapshotWithoutDuplication(t *testing.T) {
	registry := SchemaRegistry()
	schema, ok := registry[SnapshotSchemaID]
	if !ok || !json.Valid(schema) {
		t.Fatal("missing snapshot schema")
	}
	var definition map[string]any
	if err := json.Unmarshal(schema, &definition); err != nil {
		t.Fatal(err)
	}
	if definition["$id"] != SnapshotSchemaID || definition["type"] != "object" {
		t.Fatal("invalid schema identity")
	}
	for _, cap := range Catalog() {
		if cap.Mutation {
			var output map[string]any
			json.Unmarshal(cap.OutputSchema, &output)
			if output["$ref"] != SnapshotSchemaID {
				t.Fatal("mutation snapshot repeats or loses schema")
			}
		}
	}
	data, _ := json.Marshal(Catalog())
	if len(data) > 64*1024 {
		t.Fatalf("discovery inflated to %d bytes", len(data))
	}
	// Every local reference in discovery and the registry must resolve without
	// network access, including captured metadata shared by queries and snapshots.
	var check func(any)
	check = func(value any) {
		switch v := value.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok {
				if _, exists := registry[ref]; !exists {
					t.Fatalf("unresolved schema %s", ref)
				}
			}
			for _, child := range v {
				check(child)
			}
		case []any:
			for _, child := range v {
				check(child)
			}
		}
	}
	for _, schema := range registry {
		var v any
		if err := json.Unmarshal(schema, &v); err != nil {
			t.Fatal(err)
		}
		check(v)
	}
	var catalog any
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	check(catalog)

	registry[SnapshotSchemaID][0] = '!'
	if !json.Valid(SchemaRegistry()[SnapshotSchemaID]) {
		t.Fatal("client mutated registry")
	}
}
