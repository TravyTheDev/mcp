package humans

import (
	"context"
	"encoding/json"
)

type GetHumansTool struct {
	store *HumanStore
}

func NewGetHumansTool(store *HumanStore) *GetHumansTool {
	return &GetHumansTool{
		store: store,
	}
}

func (t *GetHumansTool) Name() string {
	return "get_humans"
}

func (t *GetHumansTool) Description() string {
	return "Returns all humans from the database"
}

func (t *GetHumansTool) InputSchema() any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *GetHumansTool) Call(ctx context.Context, input json.RawMessage) (any, error) {
	humans, err := t.store.GetHumans()
	if err != nil {
		return nil, err
	}
	return humans, nil
}
