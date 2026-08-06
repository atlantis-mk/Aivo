package http

import (
	"aivo/core/domain"
	"context"
	"encoding/json"
)

func (api *API) callExtensionRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "PreviewExtensionInstall":
		input, err := arg[domain.PreviewExtensionInstallInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.PreviewExtensionInstall(ctx, input)
		return result, true, err
	case "InstallExtension":
		input, err := arg[domain.InstallExtensionInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.InstallExtension(ctx, input)
		return result, true, err
	case "ListExtensionInstalls":
		result, err := api.service.ListExtensionInstalls(ctx)
		return result, true, err
	case "SetExtensionInstalledEnabled":
		input, err := arg[domain.SetExtensionEnabledInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SetExtensionInstalledEnabled(ctx, input)
		return result, true, err
	case "UninstallExtension":
		input, err := arg[domain.ExtensionControlInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		err = api.service.UninstallExtension(ctx, input)
		return map[string]any{"uninstalled": err == nil}, true, err
	case "DiscoverExtension":
		input, err := arg[domain.DiscoverExtensionInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.DiscoverExtension(ctx, input)
		return result, true, err
	case "TrustExtension":
		input, err := arg[domain.TrustExtensionInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.TrustExtension(ctx, input)
		return result, true, err
	case "EnableExtension":
		input, err := arg[domain.ExtensionControlInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.EnableExtension(ctx, input)
		return result, true, err
	case "StopExtension":
		input, err := arg[domain.ExtensionControlInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.StopExtension(ctx, input)
		return result, true, err
	case "GetExtensionStatus":
		input, err := arg[domain.ExtensionControlInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetExtensionStatus(ctx, input)
		return result, true, err
	case "ListExtensionContexts":
		input, err := arg[domain.ExtensionControlInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListExtensionContexts(ctx, input)
		return result, true, err
	case "ResolveExtensionView":
		input, err := arg[domain.ResolveExtensionViewInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ResolveExtensionView(ctx, input)
		return result, true, err
	case "OpenExtensionView":
		input, err := arg[domain.ExtensionControlInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		err = api.service.OpenExtensionView(ctx, input)
		return map[string]any{"opened": err == nil}, true, err
	case "CloseExtensionView":
		input, err := arg[domain.ExtensionControlInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		err = api.service.CloseExtensionView(ctx, input)
		return map[string]any{"closed": err == nil}, true, err
	case "InvokeExtensionViewAction":
		input, err := arg[domain.ExtensionViewActionInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.InvokeExtensionViewAction(ctx, input)
		return result, true, err
	case "BindExtensionCredential":
		input, err := arg[domain.BindExtensionCredentialInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.BindExtensionCredential(ctx, input)
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
