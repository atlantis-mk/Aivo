package http

import (
	"context"
	"encoding/json"

	"aivo/core/domain"
)

func (api *API) callPromptRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "ListPromptDocuments":
		result, err := api.service.ListPromptDocuments(ctx)
		return result, true, err
	case "GetPromptDocument":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetPromptDocument(ctx, id)
		return result, true, err
	case "ValidatePromptDraft":
		input, err := arg[domain.PromptDocumentInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ValidatePromptDraft(ctx, input)
		return result, true, err
	case "SavePromptDocument":
		input, err := arg[domain.PromptDocumentInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SavePromptDocument(ctx, input)
		return result, true, err
	case "ResetPromptDocument":
		input, err := arg[domain.PromptDocumentIDInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ResetPromptDocument(ctx, input.ID)
		return result, true, err
	case "SetPromptDocumentEnabled":
		input, err := arg[domain.PromptEnabledInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SetPromptDocumentEnabled(ctx, input)
		return result, true, err
	case "DeletePromptDocument":
		input, err := arg[domain.PromptDocumentIDInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		return nil, true, api.service.DeletePromptDocument(ctx, input.ID)
	case "ReloadPromptCatalog":
		result, err := api.service.ReloadPromptCatalog(ctx)
		return result, true, err
	case "PromptDirectory":
		result, err := api.service.PromptDirectory(ctx)
		return result, true, err
	case "CreateAgentPrompt":
		input, err := arg[domain.CreateAgentPromptInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CreateAgentPrompt(ctx, input)
		return result, true, err
	case "CreateQuickPrompt":
		input, err := arg[domain.CreateQuickPromptInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CreateQuickPrompt(ctx, input)
		return result, true, err
	case "ListPromptToolDescriptions":
		result, err := api.service.ListPromptToolDescriptions(ctx)
		return result, true, err
	default:
		return nil, false, nil
	}
}
