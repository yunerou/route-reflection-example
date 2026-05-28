package qdrant

import (
	"context"

	"github.com/google/uuid"
	"github.com/yunerou/niarb/shared/aerror"
)

// QdrantProvider defines the interface for interacting with a Qdrant vector database instance.
// It provides collection management (create, verify, query), alias management (create, resolve, swap),
// and vector operations (search, upsert).
//
// Use [New] to create an instance from a [configprovider.QdrantT] config.
//
//	provider := qdrant.New(cfg.Qdrant)
type QdrantProvider interface {
	// CollectionExists checks whether a collection with the given name exists in Qdrant.
	// Returns true if the collection exists, false otherwise.
	//
	//	exists, err := provider.CollectionExists(ctx, "documents_v1")
	CollectionExists(ctx context.Context, name string) (bool, error)

	// EnsureAlias ensures that the given alias points to the specified collection.
	// If the alias does not exist, it creates a new one.
	// If the alias already exists but points to a different collection, it returns an error.
	// If the alias already points to the correct collection, it is a no-op.
	//
	//	err := provider.EnsureAlias(ctx, "documents", "documents_v1")
	EnsureAlias(ctx context.Context, alias string, collection string) error

	// EnsureCollection ensures a collection exists with the configuration described in spec.
	// If the collection does not exist, it creates one using spec.PhysicalName, spec.VectorSize,
	// spec.Distance, and spec.OnDiskPayload.
	// If the collection already exists, it verifies that the existing configuration matches
	// the spec (vector size, distance, on_disk_payload, payload indexes).
	//
	//	err := provider.EnsureCollection(ctx, qdrant.CollectionSpec{
	//	    Alias:         "documents",
	//	    PhysicalName:  "documents_v1",
	//	    VectorSize:    1536,
	//	    Distance:      qdrant.DistanceCosine,
	//	    OnDiskPayload: true,
	//	})
	EnsureCollection(ctx context.Context, spec CollectionSpec) error

	// EnsurePayloadIndex ensures a payload index exists on the given field within a collection.
	// If the index already exists, it is a no-op.
	// If the index does not exist, it creates one and waits for completion.
	//
	//	err := provider.EnsurePayloadIndex(ctx, "documents_v1", qdrant.PayloadIndexSpec{
	//	    FieldName:   "tenant_id",
	//	    FieldSchema: "keyword",
	//	})
	EnsurePayloadIndex(ctx context.Context, collection string, spec PayloadIndexSpec) error

	// GetCollection retrieves the full details of a collection, including its vector configuration
	// (size, distance), on_disk_payload setting, and payload schema (indexed fields).
	//
	//	details, err := provider.GetCollection(ctx, "documents_v1")
	//	fmt.Println(details.Config.Params.Vectors.Size)     // e.g. 1536
	//	fmt.Println(details.Config.Params.Vectors.Distance) // e.g. "Cosine"
	GetCollection(ctx context.Context, name string) (*collectionDetails, error)

	// ResolveAlias looks up which collection an alias currently points to.
	// Returns (collectionName, true, nil) if the alias exists,
	// or ("", false, nil) if the alias is not found.
	//
	//	collectionName, found, err := provider.ResolveAlias(ctx, "documents")
	//	if found {
	//	    fmt.Println("alias points to:", collectionName)
	//	}
	ResolveAlias(ctx context.Context, alias string) (string, bool, error)

	// Search performs a vector similarity search on the specified collection.
	// Returns the top-K matches whose similarity score is >= threshold, sorted descending by score.
	// Each result includes the document UUID, similarity score, and stored payload.
	//
	//	matches, aerr := provider.Search(ctx, "documents", embeddingVector, 10, 0.7)
	//	for _, m := range matches {
	//	    fmt.Printf("doc=%s score=%.3f\n", m.DocID, m.Similarity)
	//	}
	Search(ctx context.Context, collection string, vector []float32, topK int, threshold float64) ([]VectorMatch, aerror.AError)

	// SwapAlias atomically switches an alias from one collection to another.
	// It verifies that the alias currently points to fromCollection before swapping.
	// If the alias already points to toCollection, it is a no-op.
	// If the alias points to a collection other than fromCollection, it returns an error.
	//
	// Typical use: blue-green deployment of a re-indexed collection.
	//
	//	err := provider.SwapAlias(ctx, "documents", "documents_v1", "documents_v2")
	SwapAlias(ctx context.Context, alias string, fromCollection string, toCollection string) error

	// Upsert inserts or updates a single point (vector + payload) in the specified collection.
	// The point is identified by docID (UUID). The operation waits for the write to be acknowledged.
	//
	//	aerr := provider.Upsert(ctx, "documents", docID, embeddingVector, map[string]any{
	//	    "tenant_id": "tenant-abc",
	//	    "title":     "My Document",
	//	})
	Upsert(ctx context.Context, collection string, docID uuid.UUID, vector []float32, payload map[string]any) aerror.AError

	// VerifyAliasExists checks that the given alias exists and points to an existing collection.
	// Returns an error if the alias is not found, or if it points to a collection that no longer exists.
	//
	//	err := provider.VerifyAliasExists(ctx, "documents")
	VerifyAliasExists(ctx context.Context, alias string) error

	// VerifyCollection checks that an existing collection's configuration matches the given spec.
	// Validates vector size, distance metric, on_disk_payload, and the presence of all required
	// payload indexes. Returns an error describing the first mismatch found.
	//
	//	err := provider.VerifyCollection(ctx, "documents_v1", spec)
	VerifyCollection(ctx context.Context, name string, spec CollectionSpec) error
}
