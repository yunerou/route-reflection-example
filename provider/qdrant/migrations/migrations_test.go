package migrations

import "testing"

func TestLatestVersion(t *testing.T) {
	t.Parallel()

	if LatestVersion() != 1 {
		t.Fatalf("expected latest version 1, got %d", LatestVersion())
	}
}

func TestDefaultKnowledgeCollectionSpec(t *testing.T) {
	t.Parallel()

	spec := DefaultKnowledgeCollectionSpec("knowledge", 1536)
	if spec.PhysicalName != "knowledge_v1" {
		t.Fatalf("unexpected physical name: %s", spec.PhysicalName)
	}
	if spec.VectorSize != 1536 {
		t.Fatalf("unexpected vector size: %d", spec.VectorSize)
	}
	if len(spec.PayloadIndexes) == 0 {
		t.Fatal("expected default payload indexes")
	}
}
