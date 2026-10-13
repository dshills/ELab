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
	registry[SnapshotSchemaID][0] = '!'
	if !json.Valid(SchemaRegistry()[SnapshotSchemaID]) {
		t.Fatal("client mutated registry")
	}
}
