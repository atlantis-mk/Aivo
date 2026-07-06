# Model Provider Architecture

## Provider-Neutral Contract

The agent loop depends on `domain.ChatRequest` and `domain.ChatResponse`, not provider payloads.

Core types:

- `Message` / `ChatMessage`
- `ChatRequest`
- `ChatResponse`
- `ToolSpec`
- `ToolCall`
- `ToolResult`
- `StreamingEvent`
- future usage and cost metadata
- future model capability metadata

## Provider Adapters

Adapters convert neutral messages and tools to provider-specific shapes:

- OpenAI Responses: input items, function tools, function call outputs.
- OpenAI-compatible chat completions: messages, tools, tool calls, role `tool`.
- Anthropic: content blocks, tool_use, tool_result.
- Gemini: function declarations and function responses.
- Local models: native tools when supported, prompt-based fallback otherwise.

Adapters must normalize native and streamed tool call deltas into `domain.ChatToolCall`. They must preserve `tool_call_id` correspondence and never convert tool results into user messages.

## Capability Handling

Some models support native tool calling; some do not. Provider metadata should eventually advertise native tools, parallel tool calls, streaming tool deltas, JSON schema strictness, reasoning controls, context window, usage, and cost. The agent loop can then choose native tool calls or a prompt-based fallback.

## Routing

Future model routing can select models by mode: planner, builder, reviewer, researcher, title generation, summarization, or compression. Routing belongs in app-layer provider selection, not in individual tools.
