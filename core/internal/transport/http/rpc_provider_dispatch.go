package http

import (
	"context"
	"encoding/json"

	"aivo/core/domain"
)

func (api *API) callProviderRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "GetAppConfig":
		result, err := api.service.AppConfig(ctx)
		return result, true, err
	case "GetProviderCatalog":
		result, err := api.service.Catalog(ctx)
		return result, true, err
	case "RefreshProviderEcosystemCatalog":
		input, err := arg[domain.ProviderEcosystemRefreshInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.RefreshProviderEcosystemCatalog(ctx, input)
		return result, true, err
	case "GetProviderCatalogForProject":
		projectPath, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CatalogForProject(ctx, projectPath)
		return result, true, err
	case "ConnectProvider":
		input, err := arg[domain.ProviderConnectInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ConnectProvider(ctx, input)
		return result, true, err
	case "SaveProvider":
		input, err := arg[domain.ProviderConnectInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SaveProvider(ctx, input)
		return result, true, err
	case "UpdateModelPreferences":
		input, err := arg[domain.ModelPreferencesInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		cfg, err := api.service.UpdateModelPreferences(ctx, input)
		if err == nil {
			api.events.emit("config.changed", cfg)
		}
		return cfg, true, err
	case "RefreshProviderModels":
		input, err := arg[domain.ProviderConnectInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.RefreshProviderModels(ctx, input)
		return result, true, err
	case "ValidateProvider":
		input, err := arg[domain.ProviderConnectInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ValidateProvider(ctx, input)
		return result, true, err
	case "CheckProviderIntegration":
		input, err := arg[domain.ProviderIntegrationCheckInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CheckProviderIntegration(ctx, input)
		return result, true, err
	case "ListProviderCallEvents":
		input, err := arg[domain.ProviderCallEventsInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListProviderCallEvents(ctx, input)
		return result, true, err
	case "GetProviderUsage":
		input, err := arg[domain.ProviderUsageInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetProviderUsage(ctx, input)
		return result, true, err
	case "DeleteProvider":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.DeleteProvider(ctx, id)
		return result, true, err
	case "DeleteProviderAccount":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.DeleteProviderAccount(ctx, id)
		return result, true, err
	case "StartProviderAuth":
		input, err := arg[domain.ProviderAuthStartInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.StartProviderAuth(ctx, input)
		return result, true, err
	case "GetProviderAuthStatus":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetProviderAuthStatus(ctx, id)
		return result, true, err
	case "CancelProviderAuth":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CancelProviderAuth(ctx, id)
		return result, true, err
	case "CompleteInitialization":
		input, err := arg[domain.CompleteInitializationInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.CompleteInitialization(ctx, input)
		if err == nil {
			api.events.emit("config.changed", result)
		}
		return result, true, err
	default:
		return nil, false, nil
	}
}
