package http

import (
	"context"
	"encoding/json"

	"aivo/core/domain"
)

func (api *API) callSessionEventRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "ArchiveSession":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ArchiveRuntimeSession(ctx, id)
		return result, true, err
	case "DeleteSession":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.DeleteRuntimeSession(ctx, id)
		return result, true, err
	case "GetLatestSession":
		result, err := api.service.ContinueLastSession(ctx)
		return result, true, err
	case "GetLatestSessionByProject":
		path, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ContinueProjectSession(ctx, path)
		return result, true, err
	case "ListSessionEvents":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		includeNonNormal, err := arg[bool](args, 1)
		if err != nil {
			return nil, true, err
		}
		limit, err := arg[int](args, 2)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListEvents(ctx, sessionID, includeNonNormal, limit)
		return result, true, err
	case "GetSessionRuntimeStats":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetSessionRuntimeStats(ctx, sessionID)
		return result, true, err
	case "AppendSessionEvent":
		input, err := arg[domain.AppendEventRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.AppendEvent(ctx, input)
		return result, true, err
	case "UpdateSessionEvent":
		input, err := arg[domain.UpdateSessionEventRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.UpdateSessionEvent(ctx, input)
		return result, true, err
	case "DeleteSessionEvent":
		input, err := arg[domain.DeleteSessionEventRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.DeleteSessionEvent(ctx, input)
		return result, true, err
	default:
		return nil, false, nil
	}
}
