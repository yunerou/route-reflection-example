package qdrant

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type collectionExistsResult struct {
	Exists bool `json:"exists"`
}

type collectionDetails struct {
	Config        collectionConfig         `json:"config"`
	PayloadSchema map[string]payloadSchema `json:"payload_schema"`
}

type collectionConfig struct {
	Params   collectionParams `json:"params"`
	Metadata map[string]any   `json:"metadata"`
}

type collectionParams struct {
	Vectors       vectorParams `json:"vectors"`
	OnDiskPayload bool         `json:"on_disk_payload"`
}

type vectorParams struct {
	Size     int      `json:"size"`
	Distance Distance `json:"distance"`
}

type payloadSchema struct {
	DataType any `json:"data_type"`
}

type createCollectionRequest struct {
	Vectors       vectorParams   `json:"vectors"`
	OnDiskPayload bool           `json:"on_disk_payload"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type listAliasesResult struct {
	Aliases []aliasInfo `json:"aliases"`
}

type aliasInfo struct {
	AliasName      string `json:"alias_name"`
	CollectionName string `json:"collection_name"`
}

type createPayloadIndexRequest struct {
	FieldName   string `json:"field_name"`
	FieldSchema string `json:"field_schema,omitempty"`
}

type aliasActionRequest struct {
	Actions []map[string]any `json:"actions"`
}

func (c *client) CollectionExists(ctx context.Context, name string) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("qdrant: collection name must not be empty")
	}

	var response collectionExistsResult
	err := c.doJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/collections/%s/exists", url.PathEscape(name)),
		nil,
		nil,
		&response,
	)
	if err != nil {
		return false, err
	}
	return response.Exists, nil
}

func (c *client) GetCollection(ctx context.Context, name string) (*collectionDetails, error) {
	if name == "" {
		return nil, fmt.Errorf("qdrant: collection name must not be empty")
	}

	var response collectionDetails
	err := c.doJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/collections/%s", url.PathEscape(name)),
		nil,
		nil,
		&response,
	)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *client) EnsureCollection(ctx context.Context, spec CollectionSpec) error {
	if err := spec.Validate(); err != nil {
		return err
	}

	exists, err := c.CollectionExists(ctx, spec.PhysicalName)
	if err != nil {
		return err
	}
	if !exists {
		return c.createCollection(ctx, spec)
	}
	return c.VerifyCollection(ctx, spec.PhysicalName, spec)
}

func (c *client) VerifyCollection(ctx context.Context, name string, spec CollectionSpec) error {
	if err := spec.Validate(); err != nil {
		return err
	}

	details, err := c.GetCollection(ctx, name)
	if err != nil {
		return err
	}
	if details.Config.Params.Vectors.Size != spec.VectorSize {
		return fmt.Errorf(
			"qdrant: collection %s has vector size %d, expected %d",
			name, details.Config.Params.Vectors.Size, spec.VectorSize,
		)
	}
	if details.Config.Params.Vectors.Distance != spec.Distance {
		return fmt.Errorf(
			"qdrant: collection %s has distance %s, expected %s",
			name, details.Config.Params.Vectors.Distance, spec.Distance,
		)
	}
	if details.Config.Params.OnDiskPayload != spec.OnDiskPayload {
		return fmt.Errorf(
			"qdrant: collection %s has on_disk_payload=%t, expected %t",
			name, details.Config.Params.OnDiskPayload, spec.OnDiskPayload,
		)
	}

	for _, index := range spec.PayloadIndexes {
		if _, exists := details.PayloadSchema[index.FieldName]; !exists {
			return fmt.Errorf("qdrant: collection %s is missing payload index %q", name, index.FieldName)
		}
	}

	return nil
}

func (c *client) EnsurePayloadIndex(ctx context.Context, collection string, spec PayloadIndexSpec) error {
	if collection == "" {
		return fmt.Errorf("qdrant: collection name must not be empty")
	}
	if spec.FieldName == "" {
		return fmt.Errorf("qdrant: payload index field name must not be empty")
	}

	details, err := c.GetCollection(ctx, collection)
	if err != nil {
		return err
	}
	if _, exists := details.PayloadSchema[spec.FieldName]; exists {
		return nil
	}

	query := url.Values{}
	query.Set("wait", "true")

	return c.doJSON(
		ctx,
		http.MethodPut,
		fmt.Sprintf("/collections/%s/index", url.PathEscape(collection)),
		query,
		createPayloadIndexRequest{
			FieldName:   spec.FieldName,
			FieldSchema: spec.FieldSchema,
		},
		nil,
	)
}

func (c *client) ResolveAlias(ctx context.Context, alias string) (string, bool, error) {
	if alias == "" {
		return "", false, fmt.Errorf("qdrant: alias must not be empty")
	}

	var response listAliasesResult
	err := c.doJSON(ctx, http.MethodGet, "/aliases", nil, nil, &response)
	if err != nil {
		return "", false, err
	}

	for _, item := range response.Aliases {
		if item.AliasName == alias {
			return item.CollectionName, true, nil
		}
	}
	return "", false, nil
}

func (c *client) EnsureAlias(ctx context.Context, alias string, collection string) error {
	if alias == "" {
		return fmt.Errorf("qdrant: alias must not be empty")
	}
	if collection == "" {
		return fmt.Errorf("qdrant: collection name must not be empty")
	}

	current, found, err := c.ResolveAlias(ctx, alias)
	if err != nil {
		return err
	}
	if !found {
		return c.updateAliases(ctx, []map[string]any{
			{
				"create_alias": map[string]any{
					"alias_name":      alias,
					"collection_name": collection,
				},
			},
		})
	}
	if current != collection {
		return fmt.Errorf("qdrant: alias %s already points to %s, expected %s", alias, current, collection)
	}
	return nil
}

func (c *client) VerifyAliasExists(ctx context.Context, alias string) error {
	if alias == "" {
		return fmt.Errorf("qdrant: alias must not be empty")
	}

	current, found, err := c.ResolveAlias(ctx, alias)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("qdrant: alias %s does not exist", alias)
	}

	exists, err := c.CollectionExists(ctx, current)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("qdrant: alias %s points to missing collection %s", alias, current)
	}
	return nil
}

func (c *client) SwapAlias(ctx context.Context, alias string, fromCollection string, toCollection string) error {
	if alias == "" {
		return fmt.Errorf("qdrant: alias must not be empty")
	}
	if fromCollection == "" || toCollection == "" {
		return fmt.Errorf("qdrant: swap alias collections must not be empty")
	}

	current, found, err := c.ResolveAlias(ctx, alias)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("qdrant: alias %s does not exist", alias)
	}
	if current == toCollection {
		return nil
	}
	if current != fromCollection {
		return fmt.Errorf("qdrant: alias %s points to %s, expected %s before swap", alias, current, fromCollection)
	}

	return c.updateAliases(ctx, []map[string]any{
		{
			"delete_alias": map[string]any{
				"alias_name": alias,
			},
		},
		{
			"create_alias": map[string]any{
				"alias_name":      alias,
				"collection_name": toCollection,
			},
		},
	})
}

func (c *client) createCollection(ctx context.Context, spec CollectionSpec) error {
	query := url.Values{}
	query.Set("timeout", "30")

	return c.doJSON(
		ctx,
		http.MethodPut,
		fmt.Sprintf("/collections/%s", url.PathEscape(spec.PhysicalName)),
		query,
		createCollectionRequest{
			Vectors:       vectorParams{Size: spec.VectorSize, Distance: spec.Distance},
			OnDiskPayload: spec.OnDiskPayload,
			Metadata:      spec.Metadata,
		},
		nil,
	)
}

func (c *client) updateAliases(ctx context.Context, actions []map[string]any) error {
	query := url.Values{}
	query.Set("timeout", "30")

	return c.doJSON(
		ctx,
		http.MethodPost,
		"/collections/aliases",
		query,
		aliasActionRequest{Actions: actions},
		nil,
	)
}
