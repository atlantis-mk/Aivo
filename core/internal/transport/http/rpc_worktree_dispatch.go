package http

import (
	"context"
	"encoding/json"

	"aivo/core/domain"
)

func (api *API) callWorktreeRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "CreateGitWorktree":
		input, err := arg[domain.CreateGitWorktreeInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CreateGitWorktree(ctx, input)
		return result, true, err
	case "ListGitWorktrees":
		input, err := arg[domain.ListGitWorktreesInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListGitWorktrees(ctx, input)
		return result, true, err
	case "ResetGitWorktree":
		input, err := arg[domain.ResetGitWorktreeInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ResetGitWorktree(ctx, input)
		return result, true, err
	case "RemoveGitWorktree":
		input, err := arg[domain.RemoveGitWorktreeInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.RemoveGitWorktree(ctx, input)
		return result, true, err
	case "BindSessionToGitWorktree":
		input, err := arg[domain.BindSessionGitWorktreeInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.BindSessionToGitWorktree(ctx, input)
		return result, true, err
	default:
		return nil, false, nil
	}
}
