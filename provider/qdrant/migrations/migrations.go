package migrations

import (
	"fmt"

	qdrant "github.com/yunerou/niarb/provider/qdrant"
)

func All(alias string, vectorSize int) []qdrant.Migration {
	spec := DefaultKnowledgeCollectionSpec(alias, vectorSize)
	return []qdrant.Migration{
		knowledgeInit(spec),
	}
}

func LatestVersion() int {
	return 1
}

func DefaultKnowledgeCollectionSpec(alias string, vectorSize int) qdrant.CollectionSpec {
	return qdrant.CollectionSpec{
		Alias:         alias,
		PhysicalName:  fmt.Sprintf("%s_v1", alias),
		VectorSize:    vectorSize,
		Distance:      qdrant.DistanceCosine,
		OnDiskPayload: true,
		Metadata: map[string]any{
			"logical_alias":    alias,
			"managed_by":       "niarb",
			"schema_version":   LatestVersion(),
			"vector_distance":  string(qdrant.DistanceCosine),
			"vector_dimension": vectorSize,
		},
		PayloadIndexes: []qdrant.PayloadIndexSpec{
			{FieldName: "doc_id", FieldSchema: "keyword"},
			{FieldName: "base_knowledge_id", FieldSchema: "keyword"},
			{FieldName: "mime_type", FieldSchema: "keyword"},
		},
	}
}
