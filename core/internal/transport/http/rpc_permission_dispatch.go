package http

import (
	"aivo/core/domain"
	"context"
	"encoding/json"
)

func (api *API) callPermissionRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "ListPermissionRequests":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		status, err := arg[string](args, 1)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListPermissionRequests(ctx, sessionID, status)
		return result, true, err
	case "GetPermissionMode":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetPermissionMode(ctx, sessionID)
		return result, true, err
	case "SetPermissionMode":
		input, err := arg[domain.PermissionModeInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SetPermissionMode(ctx, input)
		return result, true, err
	case "ApprovePermissionRequest":
		input, err := arg[domain.ApprovePermissionRequestInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ApprovePermissionRequest(ctx, input)
		return result, true, err
	case "DenyPermissionRequest":
		input, err := arg[domain.DenyPermissionRequestInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.DenyPermissionRequest(ctx, input)
		return result, true, err
	case "ListQuestionRequests":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		status, err := arg[string](args, 1)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListQuestionRequests(ctx, sessionID, status)
		return result, true, err
	case "ReplyQuestionRequest":
		input, err := arg[domain.ReplyQuestionRequestInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ReplyQuestionRequest(ctx, input)
		return result, true, err
	case "RejectQuestionRequest":
		input, err := arg[domain.RejectQuestionRequestInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.RejectQuestionRequest(ctx, input)
		return result, true, err
	default:
		return nil, false, nil
	}
}
