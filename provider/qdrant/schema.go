package qdrant

import (
	"fmt"

	"github.com/google/uuid"
)

// VectorMatch represents a single hit returned by vector search.
type VectorMatch struct {
	DocID      uuid.UUID
	Similarity float64
	Payload    map[string]any
}

type Distance string

const (
	DistanceCosine    Distance = "Cosine"
	DistanceDot       Distance = "Dot"
	DistanceEuclid    Distance = "Euclid"
	DistanceManhattan Distance = "Manhattan"
)

type PayloadIndexSpec struct {
	FieldName   string
	FieldSchema string
}

type CollectionSpec struct {
	Alias          string
	PhysicalName   string
	VectorSize     int
	Distance       Distance
	OnDiskPayload  bool
	Metadata       map[string]any
	PayloadIndexes []PayloadIndexSpec
}

func (s CollectionSpec) Validate() error {
	if s.Alias == "" {
		return fmt.Errorf("qdrant: collection alias must not be empty")
	}
	if s.PhysicalName == "" {
		return fmt.Errorf("qdrant: physical collection name must not be empty")
	}
	if s.VectorSize <= 0 {
		return fmt.Errorf("qdrant: vector size must be greater than 0")
	}
	if s.Distance == "" {
		return fmt.Errorf("qdrant: distance must not be empty")
	}

	seenFields := make(map[string]struct{}, len(s.PayloadIndexes))
	for _, index := range s.PayloadIndexes {
		if index.FieldName == "" {
			return fmt.Errorf("qdrant: payload index field name must not be empty")
		}
		if _, exists := seenFields[index.FieldName]; exists {
			return fmt.Errorf("qdrant: duplicated payload index field %q", index.FieldName)
		}
		seenFields[index.FieldName] = struct{}{}
	}
	return nil
}
