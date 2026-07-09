package http

import (
	"aivo/core/domain"
	"context"
	"encoding/json"
)

func (api *API) callTurnRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "StartSessionTurn":
		input, err := arg[domain.StartTurnRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.StartTurn(ctx, input)
		return result, true, err
	case "CompleteSessionTurn":
		input, err := arg[domain.CompleteTurnRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CompleteTurn(ctx, input)
		return result, true, err
	case "FailSessionTurn":
		input, err := arg[domain.FailTurnRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.FailTurn(ctx, input)
		return result, true, err
	case "CancelSessionTurn":
		input, err := arg[domain.CancelTurnRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CancelTurn(ctx, input)
		return result, true, err
	case "RetrySessionTurn":
		input, err := arg[domain.RetrySessionTurnRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.RetrySessionTurnStreaming(ctx, input)
		return result, true, err
	case "GetSessionExecutionState":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetSessionExecutionState(ctx, sessionID)
		return result, true, err
	case "InterruptSessionExecution":
		input, err := arg[domain.InterruptSessionExecutionInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.InterruptSessionExecution(ctx, input)
		return result, true, err
	case "ResumeSessionExecution":
		input, err := arg[domain.ResumeSessionExecutionInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ResumeSessionExecution(ctx, input)
		return result, true, err
	case "CompactSessionContext":
		input, err := arg[domain.CompactSessionContextInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CompactSessionContext(ctx, input)
		return result, true, err
	case "ListSessionEventsAfterCursor":
		input, err := arg[domain.ListSessionEventsAfterCursorInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListSessionEventsAfterCursor(ctx, input)
		return result, true, err
	case "GetSessionTurnDiff":
		input, err := arg[domain.GetSessionTurnDiffRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetSessionTurnDiff(ctx, input)
		return result, true, err
	case "ApplySessionTurnFileState":
		input, err := arg[domain.ApplySessionTurnFileStateRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ApplySessionTurnFileState(ctx, input)
		return result, true, err
	default:
		return nil, false, nil
	}
}
