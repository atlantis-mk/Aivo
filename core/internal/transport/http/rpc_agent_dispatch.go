package http

import (
	"context"
	"encoding/json"

	"aivo/core/domain"
)

func (api *API) callAgentRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "ListAgentModes":
		includeHidden, err := arg[bool](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListAgentModes(ctx, includeHidden)
		return result, true, err
	case "SetSessionAgentMode":
		input, err := arg[domain.SetSessionAgentModeInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SetSessionAgentMode(ctx, input)
		return result, true, err
	case "ListAgentRuns":
		input, err := arg[domain.AgentRunListRequest](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListAgentRuns(ctx, input)
		return result, true, err
	case "CancelAgentRun":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CancelAgentRun(ctx, id)
		return result, true, err
	case "ListTodoItems":
		input, err := arg[domain.TodoListInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListTodoItems(ctx, input)
		return result, true, err
	case "ListScheduledJobs":
		input, err := arg[domain.ScheduledJobListInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListScheduledJobs(ctx, input)
		return result, true, err
	case "SaveScheduledJob":
		input, err := arg[domain.ScheduledJobInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SaveScheduledJob(ctx, input)
		return result, true, err
	case "DeleteScheduledJob":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		err = api.service.DeleteScheduledJob(ctx, id)
		return nil, true, err
	case "RunDueScheduledJobs":
		limit, err := arg[int](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.RunDueScheduledJobs(ctx, limit)
		return result, true, err
	default:
		return nil, false, nil
	}
}
