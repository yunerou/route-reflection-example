package migrations

import (
	"context"

	qdrant "github.com/yunerou/niarb/provider/qdrant"
)

func knowledgeInit(spec qdrant.CollectionSpec) qdrant.Migration {
	return qdrant.Migration{
		Version: 1,
		Name:    "001_knowledge_init",
		Apply: func(ctx context.Context, client qdrant.QdrantProvider) error {
			if err := client.EnsureCollection(ctx, spec); err != nil {
				return err
			}
			for _, index := range spec.PayloadIndexes {
				if err := client.EnsurePayloadIndex(ctx, spec.PhysicalName, index); err != nil {
					return err
				}
			}
			_, found, err := client.ResolveAlias(ctx, spec.Alias)
			if err != nil {
				return err
			}
			if found {
				return nil
			}
			return client.EnsureAlias(ctx, spec.Alias, spec.PhysicalName)
		},
		Verify: func(ctx context.Context, client qdrant.QdrantProvider) error {
			if err := client.VerifyCollection(ctx, spec.PhysicalName, spec); err != nil {
				return err
			}
			return client.VerifyAliasExists(ctx, spec.Alias)
		},
	}
}
