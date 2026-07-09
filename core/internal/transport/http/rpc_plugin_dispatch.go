package http

import (
	"aivo/core/domain"
	"context"
	"encoding/json"
)

func (api *API) callPluginRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "ListPlugins":
		input, err := arg[domain.PluginListInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListPlugins(ctx, input)
		return result, true, err
	case "InstallPluginFromPath":
		input, err := arg[domain.InstallPluginInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.InstallPluginFromPath(ctx, input)
		return result, true, err
	case "SetPluginEnabled":
		input, err := arg[domain.SetPluginEnabledInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SetPluginEnabled(ctx, input)
		return result, true, err
	case "ReloadPlugins":
		result, err := api.service.ReloadPlugins(ctx)
		return result, true, err
	case "ListMCPServers":
		input, err := arg[domain.MCPServerListInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListMCPServers(ctx, input)
		return result, true, err
	case "SaveMCPServer":
		input, err := arg[domain.SaveMCPServerInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SaveMCPServer(ctx, input)
		return result, true, err
	case "SetMCPServerEnabled":
		input, err := arg[domain.SetMCPServerEnabledInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SetMCPServerEnabled(ctx, input)
		return result, true, err
	case "ProbeMCPServer":
		input, err := arg[domain.MCPProbeInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ProbeMCPServer(ctx, input)
		return result, true, err
	case "GetMCPPrompt":
		input, err := arg[domain.MCPPromptGetInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetMCPPrompt(ctx, input)
		return result, true, err
	case "ReadMCPResource":
		input, err := arg[domain.MCPResourceReadInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ReadMCPResource(ctx, input)
		return result, true, err
	case "InsertMCPPromptIntoSession":
		input, err := arg[domain.InsertMCPPromptIntoSessionInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.InsertMCPPromptIntoSession(ctx, input)
		return result, true, err
	case "InsertMCPResourceIntoSession":
		input, err := arg[domain.InsertMCPResourceIntoSessionInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.InsertMCPResourceIntoSession(ctx, input)
		return result, true, err
	case "ReadMCPServerLog":
		input, err := arg[domain.MCPServerLogInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ReadMCPServerLog(ctx, input)
		return result, true, err
	case "DiscoverMCPOAuth":
		input, err := arg[domain.MCPOAuthDiscoveryInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.DiscoverMCPOAuth(ctx, input)
		return result, true, err
	case "StartMCPOAuth":
		input, err := arg[domain.MCPOAuthStartInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.StartMCPOAuth(ctx, input)
		return result, true, err
	case "GetMCPOAuthStatus":
		input, err := arg[domain.MCPOAuthStatusInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetMCPOAuthStatus(ctx, input)
		return result, true, err
	case "ListToolCatalog":
		input, err := arg[domain.ToolCatalogInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListToolCatalog(ctx, input)
		return result, true, err
	case "DescribeTool":
		input, err := arg[domain.ToolDescribeInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.DescribeTool(ctx, input)
		return result, true, err
	case "GetSessionActiveTools":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetSessionActiveTools(ctx, sessionID)
		return result, true, err
	case "SetSessionActiveTools":
		input, err := arg[domain.SessionActiveToolsInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SetSessionActiveTools(ctx, input)
		return result, true, err
	default:
		return nil, false, nil
	}
}
