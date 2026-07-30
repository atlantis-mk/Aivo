package http

import (
	"context"
	"encoding/json"

	"aivo/core/domain"
)

func (api *API) callCommandRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "ListCommandCatalog":
		input, err := arg[domain.CommandCatalogInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListCommandCatalog(ctx, input)
		return result, true, err
	case "InvokeCommand":
		input, err := arg[domain.InvokeCommandInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.InvokeCommand(ctx, input)
		return result, true, err
	case "GetEffectiveRuntimeConfig":
		projectPath, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.EffectiveRuntimeConfig(ctx, projectPath)
		return result, true, err
	default:
		return nil, false, nil
	}
}
