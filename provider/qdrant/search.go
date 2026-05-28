package qdrant

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/yunerou/niarb/shared/aerror"
)

type queryPointsRequest struct {
	Query          []float32 `json:"query"`
	Limit          int       `json:"limit"`
	ScoreThreshold float64   `json:"score_threshold,omitempty"`
	WithPayload    bool      `json:"with_payload"`
	WithVector     bool      `json:"with_vector"`
}

type queryPointsResult struct {
	Points []queryPoint `json:"points"`
}

type queryPoint struct {
	ID      any            `json:"id"`
	Score   float64        `json:"score"`
	Payload map[string]any `json:"payload"`
}

func (c *client) Search(
	ctx context.Context,
	collection string,
	vector []float32,
	topK int,
	threshold float64,
) ([]VectorMatch, aerror.AError) {
	if collection == "" {
		return nil, aerror.New(ctx, aerror.ErrInvalidInput, fmt.Errorf("qdrant: collection must not be empty"))
	}
	if len(vector) == 0 {
		return nil, aerror.New(ctx, aerror.ErrInvalidInput, fmt.Errorf("qdrant: vector must not be empty"))
	}
	if topK <= 0 {
		return nil, aerror.New(ctx, aerror.ErrInvalidInput, fmt.Errorf("qdrant: topK must be greater than 0"))
	}

	var response queryPointsResult
	err := c.doJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/collections/%s/points/query", url.PathEscape(collection)),
		nil,
		queryPointsRequest{
			Query:          vector,
			Limit:          topK,
			ScoreThreshold: threshold,
			WithPayload:    true,
			WithVector:     false,
		},
		&response,
	)
	if err != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, err)
	}

	matches := make([]VectorMatch, 0, len(response.Points))
	for _, point := range response.Points {
		docID, err := parseUUIDPointID(point.ID)
		if err != nil {
			return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, err)
		}
		matches = append(matches, VectorMatch{
			DocID:      docID,
			Similarity: point.Score,
			Payload:    point.Payload,
		})
	}

	return matches, nil
}

func parseUUIDPointID(rawID any) (uuid.UUID, error) {
	switch id := rawID.(type) {
	case string:
		parsed, err := uuid.Parse(id)
		if err != nil {
			return uuid.Nil, fmt.Errorf("qdrant: point id %q is not a valid uuid: %w", id, err)
		}
		return parsed, nil
	default:
		return uuid.Nil, fmt.Errorf("qdrant: unsupported point id type %T, expected uuid string", rawID)
	}
}
