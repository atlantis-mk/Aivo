package http

import (
	"context"
	"encoding/json"

	coreapp "aivo/core/app"
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
	case "ResolveAgentTerminalInput":
		input, err := arg[coreapp.ResolveAgentTerminalInputRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ResolveAgentTerminalInput(ctx, input)
		return result, true, err
	case "ReleaseAgentTerminalInput":
		input, err := arg[coreapp.ReleaseAgentTerminalInputRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ReleaseAgentTerminalInput(ctx, input)
		return result, true, err
	case "ListSessionTerminals":
		workspaceRoot, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		sessionID, err := arg[string](args, 1)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListSessionTerminals(ctx, workspaceRoot, sessionID)
		return result, true, err
	case "TerminateSessionTerminals":
		workspaceRoot, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		sessionID, err := arg[string](args, 1)
		if err != nil {
			return nil, true, err
		}
		return nil, true, api.service.TerminateSessionTerminals(ctx, workspaceRoot, sessionID)
	case "UpdateSessionTerminal":
		input, err := arg[coreapp.UpdateSessionTerminalRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.UpdateSessionTerminal(ctx, input)
		return result, true, err
	case "RemoveSessionTerminal":
		workspaceRoot, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		sessionID, err := arg[string](args, 1)
		if err != nil {
			return nil, true, err
		}
		processRef, err := arg[string](args, 2)
		if err != nil {
			return nil, true, err
		}
		return nil, true, api.service.RemoveSessionTerminal(ctx, workspaceRoot, sessionID, processRef)
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
