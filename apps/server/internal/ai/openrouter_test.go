package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type deadlineThenHTTPClient struct {
	delegate  HTTPClient
	failFirst int32
	calls     atomic.Int32
}

func (c *deadlineThenHTTPClient) Do(request *http.Request) (*http.Response, error) {
	if c.calls.Add(1) <= c.failFirst {
		return nil, context.DeadlineExceeded
	}
	return c.delegate.Do(request)
}

func TestOpenRouterGenerateSendsPrivateMultimodalRequest(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		require.Equal(t, "https://app.example.test", r.Header.Get("HTTP-Referer"))
		require.Equal(t, "OpenPost Test", r.Header.Get("X-Title"))
		require.Contains(t, r.Header.Get("Accept"), "application/json")
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"gen-123",
			"object":"chat.completion",
			"created":1730000000,
			"model":"openai/gpt-5.6-luna-20260709",
			"system_fingerprint":null,
			"choices":[{
				"index":0,
				"message":{"role":"assistant","content":"  A red bicycle beside a brick wall.  "},
				"finish_reason":"stop",
				"logprobs":null
			}],
			"usage":{"prompt_tokens":21,"completion_tokens":9,"total_tokens":30,"cost":0.000012}
		}`))
	}))
	defer server.Close()

	generator, err := NewOpenRouter(OpenRouterConfig{
		APIKey:      " test-api-key ",
		BaseURL:     server.URL + "/api/v1",
		HTTPClient:  server.Client(),
		HTTPReferer: "https://app.example.test",
		XTitle:      "OpenPost Test",
		Provider:    " azure/eu ",
		RequireZDR:  true,
	})
	require.NoError(t, err)

	result, err := generator.Generate(context.Background(), GenerateRequest{
		Model:        "openai/gpt-5.6-luna",
		SystemPrompt: "Write useful alternative text.",
		UserPrompt:   "Describe the attached images.",
		ResponseSchema: &JSONSchema{
			Name:        "image_description",
			Description: "One image description",
			Schema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"description"},
				"properties": map[string]any{
					"description": map[string]any{"type": "string"},
				},
			},
		},
		MaxOutputTokens: 96,
		ReasoningEffort: ReasoningEffortNone,
		Images: []Image{
			{Data: []byte{0xff, 0xd8, 0xff}, MIMEType: "image/jpeg", Detail: ImageDetailLow},
			{Data: []byte("png"), MIMEType: "image/png"},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "A red bicycle beside a brick wall.", result.Text)
	require.Equal(t, "openai/gpt-5.6-luna-20260709", result.Model)
	require.Equal(t, "gen-123", result.RequestID)
	require.Equal(t, int64(21), result.Usage.InputTokens)
	require.Equal(t, int64(9), result.Usage.OutputTokens)
	require.Equal(t, int64(30), result.Usage.TotalTokens)
	require.NotNil(t, result.Usage.CostUSD)
	require.InDelta(t, 0.000012, *result.Usage.CostUSD, 0.0000001)

	require.Equal(t, "openai/gpt-5.6-luna", received["model"])
	require.Equal(t, float64(96), received["max_completion_tokens"])
	require.Equal(t, "none", received["reasoning_effort"])
	require.Equal(t, false, received["stream"])
	require.Equal(t, map[string]any{
		"type": "json_schema",
		"json_schema": map[string]any{
			"name":        "image_description",
			"description": "One image description",
			"strict":      true,
			"schema": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"description"},
				"properties": map[string]any{
					"description": map[string]any{"type": "string"},
				},
			},
		},
	}, received["response_format"])
	require.Equal(t, map[string]any{
		"allow_fallbacks":    false,
		"data_collection":    "deny",
		"only":               []any{"azure/eu"},
		"require_parameters": true,
		"zdr":                true,
	}, received["provider"])

	messages := received["messages"].([]any)
	require.Len(t, messages, 2)
	require.Equal(t, map[string]any{
		"role":    "system",
		"content": "Write useful alternative text.",
	}, messages[0])
	userMessage := messages[1].(map[string]any)
	require.Equal(t, "user", userMessage["role"])
	content := userMessage["content"].([]any)
	require.Equal(t, map[string]any{
		"type": "text",
		"text": "Describe the attached images.",
	}, content[0])
	require.Equal(t, map[string]any{
		"type": "image_url",
		"image_url": map[string]any{
			"url":    "data:image/jpeg;base64,/9j/",
			"detail": "low",
		},
	}, content[1])
	require.Equal(t, map[string]any{
		"type": "image_url",
		"image_url": map[string]any{
			"url":    "data:image/png;base64,cG5n",
			"detail": "low",
		},
	}, content[2])
}

func TestOpenRouterGenerateStartsFreshRequestAfterSDKDeadlineWhileCallerIsActive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"gen-after-deadline",
			"object":"chat.completion",
			"created":1730000000,
			"model":"openai/gpt-5.6-luna",
			"choices":[{
				"index":0,
				"message":{"role":"assistant","content":"recovered"},
				"finish_reason":"stop"
			}]
		}`))
	}))
	t.Cleanup(server.Close)
	client := &deadlineThenHTTPClient{delegate: server.Client(), failFirst: 2}
	generator, err := NewOpenRouter(OpenRouterConfig{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: client,
		MaxRetries: 1,
	})
	require.NoError(t, err)

	result, err := generator.Generate(context.Background(), GenerateRequest{
		Model:      "openai/gpt-5.6-luna",
		UserPrompt: "Return one word.",
	})

	require.NoError(t, err)
	require.Equal(t, "recovered", result.Text)
	require.Equal(t, int32(3), client.calls.Load())
}

func TestOpenRouterGenerateSanitizesProviderErrors(t *testing.T) {
	const privateProviderBody = "do-not-expose-this-provider-body"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":400,"message":"` + privateProviderBody + `"}}`))
	}))
	defer server.Close()

	generator, err := NewOpenRouter(OpenRouterConfig{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	require.NoError(t, err)

	_, err = generator.Generate(context.Background(), GenerateRequest{
		Model:      "openai/gpt-5.6-luna",
		UserPrompt: "Describe the image.",
	})

	var providerError *ProviderError
	require.ErrorAs(t, err, &providerError)
	require.Equal(t, http.StatusBadRequest, providerError.StatusCode)
	require.Equal(t, "OpenRouter", providerError.Provider)
	require.NotContains(t, err.Error(), privateProviderBody)
	require.NotContains(t, strings.ToLower(err.Error()), "message")
}

func TestNewOpenRouterRequiresAPIKey(t *testing.T) {
	_, err := NewOpenRouter(OpenRouterConfig{})

	require.EqualError(t, err, "OpenRouter API key is required")
}

func TestOpenRouterGenerateValidatesImageBeforeRequest(t *testing.T) {
	generator, err := NewOpenRouter(OpenRouterConfig{APIKey: "test-api-key"})
	require.NoError(t, err)

	_, err = generator.Generate(context.Background(), GenerateRequest{
		Model:      "openai/gpt-5.6-luna",
		UserPrompt: "Describe the image.",
		Images:     []Image{{Data: []byte("not-an-image"), MIMEType: "text/plain"}},
	})

	require.EqualError(t, err, "AI image 1: valid image MIME type is required")
	var providerError *ProviderError
	require.False(t, errors.As(err, &providerError))
}

// A base URL on AI/ML API's host gets the gateway's attribution headers and a
// plain chat-completions body: OpenRouter's provider preferences are its own
// routing contract and are not sent elsewhere.
func TestNewOpenRouterOnAimlapiHostSendsAttributionAndPlainBody(t *testing.T) {
	var received map[string]any
	var headers http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header.Clone()
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-1","object":"chat.completion","created":1730000000,"model":"openai/gpt-5.6-luna","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	// Route the AI/ML API hostname to the test server so the host check is real.
	client := server.Client()
	client.Transport = rewriteHostTransport{host: server.Listener.Addr().String(), inner: client.Transport}

	generator, err := NewOpenRouter(OpenRouterConfig{
		APIKey:     "test-api-key",
		BaseURL:    "https://api.aimlapi.com/v1",
		HTTPClient: client,
		Provider:   "azure/eu",
		RequireZDR: true,
	})
	require.NoError(t, err)

	_, err = generator.Generate(context.Background(), GenerateRequest{
		Model:      "openai/gpt-5.6-luna",
		UserPrompt: "hi",
	})
	require.NoError(t, err)
	require.Equal(t, "agent/openpost", headers.Get("X-AIMLAPI-Source"))
	require.NotEmpty(t, headers.Get("X-AIMLAPI-Partner-ID"))
	require.NotContains(t, received, "provider")
	require.Equal(t, false, received["stream"])
}

// The attribution headers belong to the host, not to the configuration: a
// proxy or a look-alike hostname gets none, and keeps OpenRouter's dialect.
func TestNewOpenRouterElsewhereSendsNoAimlapiHeaders(t *testing.T) {
	var headers http.Header
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header.Clone()
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"gen-1","object":"chat.completion","created":1730000000,"model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	for _, baseURL := range []string{server.URL + "/api/v1", "https://api.aimlapi.com.evil.example/v1"} {
		client := server.Client()
		client.Transport = rewriteHostTransport{host: server.Listener.Addr().String(), inner: client.Transport}
		generator, err := NewOpenRouter(OpenRouterConfig{APIKey: "k", BaseURL: baseURL, HTTPClient: client})
		require.NoError(t, err)
		_, err = generator.Generate(context.Background(), GenerateRequest{Model: "m", UserPrompt: "hi"})
		require.NoError(t, err, baseURL)
		require.Empty(t, headers.Get("X-AIMLAPI-Source"), baseURL)
		require.Empty(t, headers.Get("X-AIMLAPI-Partner-ID"), baseURL)
		require.Contains(t, received, "provider", baseURL)
	}
}

// Web search is an OpenRouter plugin; asking for it on another gateway is an
// error rather than a request that silently loses the tool.
func TestBuildOpenRouterRequestRejectsWebSearchOffOpenRouter(t *testing.T) {
	_, _, err := buildOpenRouterRequest(GenerateRequest{
		Model:      "m",
		UserPrompt: "hi",
		WebSearch:  WebSearchConfig{Enabled: true, MaxResults: 5, MaxUses: 1, MaxTotalResults: 5, MaxCharactersPerResult: 1000, Context: WebSearchContextLow},
	}, "", false, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "only available through OpenRouter")
}

// rewriteHostTransport sends every request to a fixed address while leaving
// the URL's hostname — and therefore the client's host-based decisions — intact.
type rewriteHostTransport struct {
	host  string
	inner http.RoundTripper
}

func (t rewriteHostTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	clone := r.Clone(r.Context())
	clone.URL.Scheme = "http"
	clone.URL.Host = t.host
	return t.inner.RoundTrip(clone)
}
