package http

import (
	coreapp "aivo/core/app"
	"context"
	"encoding/json"
)

func (api *API) callTerminalRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "ListTerminals":
		workspaceRoot, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListTerminals(ctx, workspaceRoot)
		return result, true, err
	case "CreateTerminal":
		input, err := arg[coreapp.TerminalCreateInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CreateTerminal(ctx, input)
		return result, true, err
	case "GetTerminal":
		workspaceRoot, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		terminalID, err := arg[string](args, 1)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetTerminal(ctx, workspaceRoot, terminalID)
		return result, true, err
	case "UpdateTerminal":
		input, err := arg[coreapp.TerminalUpdateInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.UpdateTerminal(ctx, input)
		return result, true, err
	case "RemoveTerminal":
		workspaceRoot, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		terminalID, err := arg[string](args, 1)
		if err != nil {
			return nil, true, err
		}
		return nil, true, api.service.RemoveTerminal(ctx, workspaceRoot, terminalID)
	case "PollShellProcess":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.PollShellProcess(id)
		return result, true, err
	case "WaitShellProcess":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.WaitShellProcess(ctx, id)
		return result, true, err
	case "KillShellProcess":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.KillShellProcess(id)
		return result, true, err
	case "ReadShellProcessOutput":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ReadShellProcessOutput(id)
		return result, true, err
	default:
		return nil, false, nil
	}
}
