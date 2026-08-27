# AI Gateway

A lightweight, extensible AI Gateway written in Go with no web framework — only the Go standard library and `gopkg.in/yaml.v3`.

It exposes an OpenAI-compatible Chat Completions API and routes requests to configurable LLM providers (OpenAI, DeepSeek, Ollama, or a built-in mock). It is designed as the foundation for future Agent, Tool, MCP, RAG, and enterprise AI platform capabilities.

## Features

Implemented in the current codebase (V1):

- **OpenAI-compatible Chat Completions API** — `POST /v1/chat/completions`, non-streaming and streaming.
- **API Key authentication** — clients authenticate with `Authorization: Bearer <key>`; the key is configured via the `AI_GATEWAY_API_KEY` environment variable.
- **Model routing** — the `model` requested by the client is mapped to a `(provider, provider model)` pair via `config.yaml`.
- **Provider abstraction** — all backends implement a single `ModelProvider` interface.
- **OpenAI-compatible provider** — works with any OpenAI-compatible endpoint; used for OpenAI, DeepSeek, and Ollama (via its `/v1` compatibility endpoint).
- **Mock provider** — returns deterministic responses for testing and local development without any external service.
- **Streaming (SSE)** — `stream: true` responses are delivered as Server-Sent Events with a unified `chat.completion.chunk` format, terminated by `data: [DONE]`.
- **Conversation management** — multi-turn conversations with history reuse via `conversation_id` (in-memory store).
- **Structured error handling** — consistent JSON error responses with machine-readable `type` / `code` fields.
- **Tests** — unit tests for config, conversation store, and chat/conversation services, run with the Go race detector.

## Architecture

```
                    ┌─────────────────────────────────────────────────────┐
                    │                    Gateway                          │
                    │                                                     │
 Client ──────────► │  APIKeyAuth        HTTP Router        Handler       │
 (Bearer key)       │  (middleware) ───► (net/http          (chat /       │
                    │                    ServeMux)           conversation │
                    │                                    / models /      │
                    │                                     health)        │
                    │                                         │          │
                    │                                         ▼          │
                    │                                   ChatService      │
                    │                                     │        │     │
                    │              ┌──────────────────────┘        ▼     │
                    │              ▼                    ConversationService
                    │        ModelRouter                    │            │
                    │           │       │                   ▼            │
                    │           │       │          Conversation Store    │
                    │           │       │           (in-memory)          │
                    │           ▼       ▼                                │
                    │   OpenAI-compatible    Mock                        │
                    │   Provider             Provider                    │
                    └───────────┬────────────────────────────────────────┘
                                ▼
                       Upstream LLM
                       (OpenAI / DeepSeek / Ollama)
```

Notes:

- The `openai-compatible` provider is used for OpenAI, DeepSeek, and Ollama. Ollama is reached through its OpenAI-compatible endpoint (`http://localhost:11434/v1`); there is no separate Ollama-specific provider type.
- The mock provider requires no upstream and returns fixed responses, which keeps the test suite hermetic.
- Conversations are stored only in memory (`internal/conversation/memory.go`). There is no persistent store.
- `/health` is registered without the auth middleware; all other routes (`/v1/models`, `/v1/chat/completions`, `/v1/conversations`) require the API key.

## Request Flow

### Non-streaming

```
Client
  → API Key authentication (middleware)
  → Handler (decode & validate request)
  → ChatService
  → Conversation (if conversation_id is set: load history, merge with new messages)
  → ModelRouter (resolve model → provider + provider model)
  → Provider
  → Upstream LLM
  → Provider (normalize response)
  → ChatService (persist messages if conversation_id is set)
  → Handler
  → Client (JSON response)
```

### Streaming (`"stream": true`)

```
Client
  → API Key authentication (middleware)
  → Handler (SSE headers: text/event-stream, no-cache, keep-alive)
  → ChatService.ChatStream
  → Conversation (if conversation_id is set: load history)
  → ModelRouter
  → Provider.ChatStream
  → Upstream LLM (SSE)
```

Data path for each chunk:

```
Provider
  → ChatService (accumulates assistant content)
  → StreamWriter
  → HTTP Handler (writeSSE: "data: {chunk}\n\n" + flush)
  → Client
```

The stream is terminated with `data: [DONE]`. Provider chunks are converted into the gateway's unified `ChatCompletionChunk` format before being written to the client.

## Provider Architecture

All backends implement the `ModelProvider` interface (`internal/provider/provider.go`):

```go
type ModelProvider interface {
    Chat(ctx context.Context, req *chat.ChatCompletionRequest) (*chat.ChatCompletionResponse, error)
    ChatStream(ctx context.Context, req *chat.ChatCompletionRequest, writer StreamWriter) error
}
```

The provider factory (`internal/provider/factory`) currently registers two types:

| Config `type`   | Implementation                              | Used for                          |
| --------------- | ------------------------------------------- | --------------------------------- |
| `openai-compatible` | `internal/provider/openaicompatible`   | OpenAI, DeepSeek, Ollama `/v1`    |
| `mock`          | `internal/provider/mock`                    | Testing / local development       |

Because every backend is hidden behind the same interface, the service layer (`ChatService`) never depends on a specific vendor. Adding a new backend only requires implementing the interface and registering it in the factory — no changes to handlers or services.

The provider package also keeps additional implementations as architecture foundation: an OpenAI-specific provider (`internal/provider/openai`) and a standalone mock implementation (`internal/provider/mock.go`), alongside the `openaicompatible` and `mock` providers currently registered by the factory.

## Model Routing

The `model` field in a client request does not have to match the model name used by the provider. `ModelRouter` resolves the requested model against `config.yaml`:

| Requested model | Provider | Provider model |
| --------------- | -------- | -------------- |
| `test-model`    | mock     | `test-model`   |
| `gpt-5.6`       | openai   | `gpt-5.6`      |
| `deepseek-chat` | deepseek | `deepseek-v4-pro` |
| `qwen3`         | ollama   | `qwen3:8b`     |

For example, a client sends `"model": "qwen3"`; the gateway routes it to the `ollama` provider and sends `"model": "qwen3:8b"` to the upstream. Requesting an unregistered model returns a `400` error with code `model_not_supported`.

## Conversation

A `Conversation` groups messages belonging to the same multi-turn dialogue:

1. `POST /v1/conversations` creates a conversation (ID format: `conv_<hex>`).
2. Subsequent chat requests include `conversation_id`.
3. `ChatService` loads the history from the conversation store, appends the current request messages, and sends the combined message list to the provider.
4. After a successful completion, the request messages and the assistant reply are persisted to the conversation.

The current implementation uses an **in-memory store** (`conversation.NewMemoryStore()`, protected by a `sync.RWMutex`). Conversations are lost when the process restarts. There is no Redis / MySQL / PostgreSQL persistence yet — that is roadmap work.

If `conversation_id` is set but the conversation does not exist, the request fails (the store returns "conversation not found").

## Streaming

Streaming uses Server-Sent Events. When `stream: true`:

- The handler sets `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`.
- Each chunk produced by the provider is converted into the unified `ChatCompletionChunk` JSON and written as `data: {...}\n\n`, followed by a flush.
- The stream ends with `data: [DONE]\n\n`.
- For conversations, the accumulated assistant message is persisted after the stream completes.

Note: once the stream has started, the HTTP status code can no longer be changed, so errors that occur mid-stream terminate the stream rather than producing a JSON error response.

## Authentication

The gateway has its own API key authentication. Clients must send:

```
Authorization: Bearer <AI_GATEWAY_API_KEY>
```

The key is read from the `AI_GATEWAY_API_KEY` environment variable at startup; the gateway refuses to start if it is unset. All business endpoints require this header; `/health` is exempt from authentication so that load balancers and monitors can probe the service without a key.

Two distinct credential layers exist:

1. **Gateway API key** — issued by the gateway operator, used by *clients calling the gateway*.
2. **Provider API keys** — credentials for upstream LLMs (e.g. `OPENAI_API_KEY`, `DEEPSEEK_API_KEY`), configured in `config.yaml` and used by the gateway when calling *providers*. Clients never see or send these.

## Configuration

`config.yaml` defines:

- `server` — listen host and port (`0.0.0.0:8080` by default).
- `providers` — name → `{type, base_url, api_key}`.
- `models` — requested model name → `{provider, model}`.

Environment variable expansion is applied to the config file content (`${VAR}` syntax), so provider keys are referenced, not hardcoded:

```yaml
providers:
  deepseek:
    type: openai-compatible
    base_url: https://api.deepseek.com/v1
    api_key: ${DEEPSEEK_API_KEY}
```

Environment variables:

| Variable              | Purpose                                             |
| --------------------- | --------------------------------------------------- |
| `AI_GATEWAY_API_KEY` | Gateway API key for client authentication (required) |
| `OPENAI_API_KEY`     | OpenAI provider key                                 |
| `DEEPSEEK_API_KEY`   | DeepSeek provider key                               |
| `AI_GATEWAY_CONFIG`  | Optional path to the config file (default: `config.yaml`) |

`.env.example` lists the expected variables for local setup. The application does **not** load `.env` automatically — export the variables in your shell. Real API keys, tokens, passwords, and private keys must never be committed to Git; `.gitignore` already excludes `.env`, `*.key`, `*.pem`, and `*.secret`.

## Getting Started

Prerequisites: Go (see `go.mod` for the required version) and, optionally, a running Ollama instance for local models.

```bash
# 1. Clone
git clone <repository-url>
cd ai_gateway

# 2. Configure environment variables
export AI_GATEWAY_API_KEY=your_gateway_api_key
export OPENAI_API_KEY=your_openai_api_key       # optional
export DEEPSEEK_API_KEY=your_deepseek_api_key   # optional

# 3. Review / edit config.yaml (providers and model routes)

# 4. Run
go run ./cmd/gateway
```

The gateway logs its listen address (default `http://localhost:8080`).

## API

All endpoints require `Authorization: Bearer $AI_GATEWAY_API_KEY`.

### `POST /v1/chat/completions`

Request body:

```json
{
  "model": "test-model",
  "messages": [
    { "role": "user", "content": "Hello" }
  ],
  "temperature": 0.7,
  "stream": false,
  "conversation_id": ""
}
```

| Field            | Type    | Notes                                              |
| ---------------- | ------- | -------------------------------------------------- |
| `model`          | string  | Required; must be a model registered in `config.yaml` |
| `messages`       | array   | Required; `{role, content}` items                  |
| `temperature`    | number  | Optional; forwarded to providers when set |
| `stream`         | bool    | Optional; enables SSE streaming                    |
| `conversation_id`| string  | Optional; must reference an existing conversation  |

Basic request:

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $AI_GATEWAY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test-model",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

Streaming request:

```bash
curl -N http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $AI_GATEWAY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test-model",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

Conversation request:

```bash
# 1. Create a conversation
curl http://localhost:8080/v1/conversations \
  -H "Authorization: Bearer $AI_GATEWAY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "test-model"}'
# → {"id":"conv_...","object":"conversation","model":"test-model"}

# 2. Chat within the conversation
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $AI_GATEWAY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test-model",
    "messages": [{"role": "user", "content": "Hello"}],
    "conversation_id": "conv_..."
  }'
```

### Other endpoints

| Endpoint               | Method | Description                                                        |
| ---------------------- | ------ | ------------------------------------------------------------------ |
| `/v1/conversations`    | POST   | Create a conversation (`{"model": "..."}`)                         |
| `/v1/models`           | GET    | List models (currently a fixed list: `qwen3`, `deepseek`)          |
| `/health`              | GET    | Health check (`{"status":"ok"}`); no authentication required       |

### Error format

Errors are returned as JSON:

```json
{
  "error": {
    "message": "model 'foo' is not supported",
    "type": "invalid_request",
    "code": "model_not_supported"
  }
}
```

## Project Structure

```
ai_gateway/
├── cmd/
│   └── gateway/
│       └── main.go                  # entry point: env, auth middleware, server bootstrap
├── internal/
│   ├── app/
│   │   └── app.go                   # application assembly: router, services, store
│   ├── chat/
│   │   └── chat.go                  # request/response/chunk types (OpenAI-compatible)
│   ├── config/
│   │   ├── config.go                # config structs
│   │   ├── loader.go                # YAML loading, env expansion, validation
│   │   └── config_test.go
│   ├── conversation/
│   │   ├── conversation.go          # Conversation, Message, Store interface
│   │   ├── memory.go                # in-memory store (mutex-protected)
│   │   └── memory_test.go
│   ├── errors/
│   │   └── errors.go                # gateway error type and codes
│   ├── handler/
│   │   ├── chat.go                  # chat completions + SSE handler
│   │   ├── conversation.go          # conversation creation handler
│   │   ├── models.go                # models listing handler
│   │   ├── error.go                 # JSON error writer
│   │   └── heatth.go                # health check handler
│   ├── middleware/
│   │   └── auth.go                  # API key authentication middleware
│   ├── model/
│   │   └── model.go                 # model representation for /v1/models
│   ├── modelrouter/
│   │   └── router.go                # model → (provider, provider model) resolution
│   ├── provider/
│   │   ├── provider.go              # ModelProvider interface + StreamWriter
│   │   ├── mock.go                  # standalone mock provider implementation
│   │   ├── factory/
│   │   │   └── factory.go           # provider creation by config type
│   │   ├── mock/
│   │   │   └── provider.go          # mock provider used by the factory
│   │   ├── openai/
│   │   │   └── openai.go            # OpenAI-specific provider implementation
│   │   └── openaicompatible/
│   │       ├── client.go            # HTTP client for OpenAI-compatible endpoints
│   │       ├── provider.go          # provider implementation (chat + SSE streaming)
│   │       └── error.go             # provider error mapping
│   ├── router/
│   │   └── router.go                # route registration (net/http ServeMux)
│   ├── server/
│   │   └── server.go                # HTTP server wrapper
│   └── service/
│       ├── chat.go                  # ChatService: routing, conversation merge, persistence
│       ├── conversation.go          # ConversationService
│       ├── chat_conversation_test.go
│       └── conversation_test.go
├── config.yaml                      # providers and model routes
├── .env.example                     # expected environment variables
├── .gitignore
├── go.mod
└── go.sum
```

## Testing

```bash
go test ./...        # unit tests
go test -race ./...  # unit tests with the race detector
go build ./...       # compile all packages
go fmt ./...         # format all packages
```

The test suite covers the config loader, the in-memory conversation store, and the chat/conversation services. The race detector has been used to check for concurrency issues (e.g. concurrent access to the in-memory conversation store).

## Current Status

The project is at **V1** — a functional LLM gateway.

- [x] Chat Completions API (`POST /v1/chat/completions`)
- [x] Streaming (SSE)
- [x] API Key authentication
- [x] Provider abstraction (`ModelProvider` interface)
- [x] Model routing
- [x] Conversation management (in-memory)
- [x] OpenAI-compatible provider (OpenAI / DeepSeek / Ollama)
- [x] Mock provider
- [x] Structured error handling
- [x] Unit tests + race detector

Known limitations of the current implementation:

- `/v1/models` returns a fixed list that is not derived from `config.yaml`.
- Conversation history is in-memory only; it is lost on restart.
- No graceful shutdown handling.

## Roadmap

Planned capabilities — **not yet implemented**:

**V1.x / Next — Agent Runtime**

- Agent
- LLM
- Memory
- Tool
- MCP

**Later — RAG**

- Knowledge Base
- Retrieval
- Vector Store
- Enterprise Data

**Final goal — Enterprise AI Platform**

- Agent Runtime
- RAG
- MCP
- Tool Management
- Model Management
- Authentication & Authorization
- Usage Tracking
- Observability
- Audit Logging

## Design Principles

- **Provider independence** — the service layer depends only on the `ModelProvider` interface; adding or swapping backends requires no changes above the factory.
- **Model routing** — clients reference stable model names; the actual provider and provider-side model are resolved by configuration.
- **API compatibility** — the public API follows the OpenAI Chat Completions conventions, so existing OpenAI clients can be pointed at the gateway with minimal changes.
- **Separation of concerns** — HTTP handling, business logic, routing, provider I/O, and storage live in separate packages.
- **Interface driven** — conversation storage and provider access are defined as interfaces (`conversation.Store`, `provider.ModelProvider`), keeping implementations replaceable.
- **Incremental architecture** — the gateway intentionally stays lightweight (stdlib `net/http`, no framework, no database). Complexity such as microservices, queues, or persistent stores will only be introduced when a concrete requirement demands it.

## License

License will be added before public release.
