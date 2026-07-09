package http

import (
	"aivo/core/domain"
	"context"
	"encoding/json"
)

func (api *API) callSessionContextRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "ForkSession":
		input, err := arg[domain.ForkSessionRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ForkSession(ctx, input)
		return result, true, err
	case "CreateSessionSummary":
		input, err := arg[domain.CreateSummaryRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CreateSummary(ctx, input)
		return result, true, err
	case "GetLatestSessionSummary":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.LatestSummary(ctx, id)
		return result, true, err
	case "CreateSessionCheckpoint":
		input, err := arg[domain.CreateCheckpointRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CreateCheckpoint(ctx, input)
		return result, true, err
	case "ListSessionCheckpoints":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		limit, err := arg[int](args, 1)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListCheckpoints(ctx, sessionID, limit)
		return result, true, err
	case "GetLatestSessionCheckpoint":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.LatestCheckpoint(ctx, id)
		return result, true, err
	case "GetCodingContext":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetCodingContext(ctx, id)
		return result, true, err
	case "UpdateCodingContext":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		projectPath, err := arg[string](args, 1)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CreateOrUpdateCodingContext(ctx, sessionID, projectPath)
		return result, true, err
	case "ResumeSession":
		input, err := arg[domain.ResumeSessionRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ResumeRecap(ctx, input)
		return result, true, err
	case "BuildSessionContext":
		input, err := arg[domain.BuildSessionContextRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.BuildSessionContext(ctx, input)
		return result, true, err
	case "ListSessionTurns":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		limit, err := arg[int](args, 1)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListTurns(ctx, sessionID, limit)
		return result, true, err
	case "SaveSessionToolCall":
		input, err := arg[domain.CreateToolCallRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SaveToolCall(ctx, input)
		return result, true, err
	case "ListSessionToolCalls":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListToolCalls(ctx, sessionID)
		return result, true, err
	case "ReplaySessionToolCall":
		input, err := arg[domain.ReplaySessionToolCallRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ReplaySessionToolCall(ctx, input)
		return result, true, err
	case "ReadRetainedOutput":
		input, err := arg[domain.RetainedOutputReadInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ReadRetainedOutput(ctx, input)
		return result, true, err
	default:
		return nil, false, nil
	}
}
