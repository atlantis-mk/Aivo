package http

import (
	"context"
	"encoding/json"
	"fmt"
)

func (api *API) call(ctx context.Context, method string, args []json.RawMessage) (interface{}, error) {
	if result, handled, err := api.callProviderRPC(ctx, method, args); handled {
		return result, err
	}
	if result, handled, err := api.callSessionRPC(ctx, method, args); handled {
		return result, err
	}
	if result, handled, err := api.callAgentRPC(ctx, method, args); handled {
		return result, err
	}
	if result, handled, err := api.callSessionEventRPC(ctx, method, args); handled {
		return result, err
	}
	if result, handled, err := api.callTurnRPC(ctx, method, args); handled {
		return result, err
	}
	if result, handled, err := api.callSessionContextRPC(ctx, method, args); handled {
		return result, err
	}
	if result, handled, err := api.callPermissionRPC(ctx, method, args); handled {
		return result, err
	}
	if result, handled, err := api.callTerminalRPC(ctx, method, args); handled {
		return result, err
	}
	if result, handled, err := api.callSkillRPC(ctx, method, args); handled {
		return result, err
	}
	if result, handled, err := api.callPluginRPC(ctx, method, args); handled {
		return result, err
	}
	if result, handled, err := api.callWorktreeRPC(ctx, method, args); handled {
		return result, err
	}
	if result, handled, err := api.callCommandRPC(ctx, method, args); handled {
		return result, err
	}
	return nil, fmt.Errorf("unknown RPC method %q", method)
}
