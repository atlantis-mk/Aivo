package http

import (
	"context"
	"encoding/json"

	"aivo/core/domain"
)

func (api *API) callSessionRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "SelectProjectDirectory":
		result, err := api.service.SelectProjectDirectory("")
		return result, true, err
	case "UpsertProject":
		path, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.UpsertProject(ctx, path)
		return result, true, err
	case "SetProjectSidebarHidden":
		path, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		hidden, err := arg[bool](args, 1)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SetProjectSidebarHidden(ctx, path, hidden)
		return result, true, err
	case "ListRecentProjects":
		limit, err := arg[int](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListProjects(ctx, limit)
		return result, true, err
	case "CreateSession":
		input, err := arg[domain.CreateSessionRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CreateRuntimeSession(ctx, input)
		return result, true, err
	case "ListSessions":
		input, err := arg[domain.ListSessionsRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListRuntimeSessions(ctx, input)
		return result, true, err
	case "SubmitSessionMessage", "SubmitSessionMessageStreaming":
		input, err := arg[domain.SubmitSessionMessageRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SubmitSessionMessageStreaming(ctx, input)
		return result, true, err
	case "GetSession":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetRuntimeSession(ctx, id)
		return result, true, err
	case "UpdateSession":
		input, err := arg[domain.UpdateSessionRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.UpdateRuntimeSession(ctx, input)
		return result, true, err
	default:
		return nil, false, nil
	}
}
