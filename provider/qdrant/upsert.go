package qdrant

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/yunerou/niarb/shared/aerror"
)

type upsertPointsRequest struct {
	Points []upsertPoint `json:"points"`
}

type upsertPoint struct {
	ID      string         `json:"id"`
	Vector  []float32      `json:"vector"`
	Payload map[string]any `json:"payload,omitempty"`
}

func (c *client) Upsert(
	ctx context.Context,
	collection string,
	docID uuid.UUID,
	vector []float32,
	payload map[string]any,
) aerror.AError {
	if collection == "" {
		return aerror.New(ctx, aerror.ErrInvalidInput, fmt.Errorf("qdrant: collection must not be empty"))
	}
	if docID == uuid.Nil {
		return aerror.New(ctx, aerror.ErrInvalidInput, fmt.Errorf("qdrant: docID must not be nil"))
	}
	if len(vector) == 0 {
		return aerror.New(ctx, aerror.ErrInvalidInput, fmt.Errorf("qdrant: vector must not be empty"))
	}

	query := url.Values{}
	query.Set("wait", "true")

	err := c.doJSON(
		ctx,
		http.MethodPut,
		fmt.Sprintf("/collections/%s/points", url.PathEscape(collection)),
		query,
		upsertPointsRequest{
			Points: []upsertPoint{
				{
					ID:      docID.String(),
					Vector:  vector,
					Payload: payload,
				},
			},
		},
		nil,
	)
	if err != nil {
		return aerror.New(ctx, aerror.ErrUnexpectedNetwork, err)
	}
	return nil
}
