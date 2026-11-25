package provider

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/bkelceoglu/go-rt/internal/domain"
)

var _ domain.Provider = (*OpenAICompatibleProvider)(nil)

// OpenAICompatibleProvider handles OpenAI, Ollama, vLLM, and any v1/chat/completions endpoint.
type OpenAICompatibleProvider struct {
	spec       domain.ProviderSpec
	modelName  string
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

// NewOpenAICompatible creates a ready-to-call provider from domain.ProviderSpec.
func NewOpenAICompatible(spec domain.ProviderSpec, client *http.Client) *OpenAICompatibleProvider {
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	return &OpenAICompatibleProvider{
		spec:       spec,
		modelName:  resolveModelName(spec.ID),
		endpoint:   joinCompletionsPath(resolveBaseURL(spec.ID, spec.BaseURL)),
		apiKey:     resolveAPIKey(spec.ID, spec.APIKey),
		httpClient: client,
	}
}

// ID returns the configured provider identifier.
func (p *OpenAICompatibleProvider) ID() string {
	return p.spec.ID
}

// Call executes a single prompt completion against the target model.
func (p *OpenAICompatibleProvider) Call(ctx context.Context, req domain.PromptRequest) (domain.ProviderResponse, error) {
	httpReq, err := p.prepareChatRequest(ctx, req)
	if err != nil {
		return domain.ProviderResponse{}, err
	}
	start := time.Now()
	respBytes, statusCode, err := p.executeHTTP(httpReq)
	if err != nil {
		return domain.ProviderResponse{}, err
	}
	return p.parseAPIResponse(respBytes, statusCode, time.Since(start).Milliseconds())
}

func (p *OpenAICompatibleProvider) prepareChatRequest(ctx context.Context, req domain.PromptRequest) (*http.Request, error) {
	bodyBytes, err := p.marshalRequestBody(req)
	if err != nil {
		return nil, err
	}
	return p.buildHTTPRequest(ctx, bodyBytes)
}

func (p *OpenAICompatibleProvider) marshalRequestBody(req domain.PromptRequest) ([]byte, error) {
	payload := map[string]any{
		"model":    p.modelName,
		"messages": buildOpenAIMessages(req, p.spec.Think),
	}
	if p.spec.Temp > 0 {
		payload["temperature"] = p.spec.Temp
	}
	if p.spec.MaxToks > 0 {
		payload["max_tokens"] = p.spec.MaxToks
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request body for %s: %w", p.spec.ID, err)
	}
	return data, nil
}

func buildOpenAIMessages(req domain.PromptRequest, think *bool) []domain.Message {
	if len(req.Messages) > 0 {
		msgs := slices.Clone(req.Messages)
		lastIdx := len(msgs) - 1
		if msgs[lastIdx].Role == "user" {
			msgs[lastIdx].Content = applyThinkDirective(msgs[lastIdx].Content, think)
		}
		return msgs
	}
	return []domain.Message{
		{Role: "user", Content: applyThinkDirective(req.Prompt, think)},
	}
}

func applyThinkDirective(prompt string, think *bool) string {
	if think == nil {
		return prompt
	}
	if *think {
		return "/think\n" + prompt
	}
	return "/no_think\n" + prompt
}

func (p *OpenAICompatibleProvider) buildHTTPRequest(ctx context.Context, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create http request for %s: %w", p.spec.ID, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	for k, v := range p.spec.Headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

func (p *OpenAICompatibleProvider) executeHTTP(req *http.Request) ([]byte, int, error) {
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request to %s: %w", p.spec.ID, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response from %s: %w", p.spec.ID, err)
	}
	return data, resp.StatusCode, nil
}

func (p *OpenAICompatibleProvider) parseAPIResponse(data []byte, status int, latency int64) (domain.ProviderResponse, error) {
	if status < 200 || status >= 300 {
		return domain.ProviderResponse{}, fmt.Errorf("provider %s returned http %d: %s", p.spec.ID, status, string(data))
	}
	var raw chatCompletionResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return domain.ProviderResponse{}, fmt.Errorf("decode response from %s: %w", p.spec.ID, err)
	}
	return buildProviderResponse(raw, latency), nil
}

func buildProviderResponse(raw chatCompletionResponse, latency int64) domain.ProviderResponse {
	return domain.ProviderResponse{
		Output:    extractChoiceText(raw.Choices),
		LatencyMs: latency,
		TokenUsage: domain.TokenUsage{
			Prompt:     raw.Usage.PromptTokens,
			Completion: raw.Usage.CompletionTokens,
			Total:      raw.Usage.TotalTokens,
			Requests:   1,
		},
	}
}

func extractChoiceText(choices []chatChoice) string {
	if len(choices) == 0 {
		return ""
	}
	return cmp.Or(strings.TrimSpace(choices[0].Message.Content), strings.TrimSpace(choices[0].Message.Reasoning))
}

var knownModelPrefixes = []string{
	"llamacpp:chat:", "llamacpp:",
	"llama:chat:", "llama:",
	"localai:chat:", "localai:",
	"openai:chat:", "openai:",
	"ollama:chat:", "ollama:",
	"vllm:chat:", "vllm:",
}

func resolveModelName(id string) string {
	for _, p := range knownModelPrefixes {
		if after, ok := strings.CutPrefix(id, p); ok {
			return after
		}
	}
	return id
}

func resolveBaseURL(id, explicit string) string {
	if url := cmp.Or(explicit, resolveEnvBaseURL(id)); url != "" {
		return strings.TrimRight(url, "/")
	}
	return defaultBaseURL(id)
}

func resolveEnvBaseURL(id string) string {
	switch {
	case strings.HasPrefix(id, "llamacpp") || strings.HasPrefix(id, "llama"):
		return os.Getenv("LLAMACPP_BASE_URL")
	case strings.HasPrefix(id, "localai"):
		return os.Getenv("LOCALAI_BASE_URL")
	case strings.HasPrefix(id, "ollama"):
		return cmp.Or(os.Getenv("OLLAMA_BASE_URL"), os.Getenv("OLLAMA_HOST"))
	case strings.HasPrefix(id, "openai"):
		return os.Getenv("OPENAI_BASE_URL")
	case strings.HasPrefix(id, "vllm"):
		return os.Getenv("VLLM_BASE_URL")
	default:
		return os.Getenv("LLAMACPP_BASE_URL")
	}
}

func defaultBaseURL(id string) string {
	switch {
	case strings.HasPrefix(id, "openai"):
		return "https://api.openai.com/v1"
	case strings.HasPrefix(id, "localai"):
		return "http://localhost:8090/v1"
	case strings.HasPrefix(id, "vllm"):
		return "http://localhost:8000/v1"
	case strings.HasPrefix(id, "ollama"):
		return "http://localhost:11434/v1"
	default:
		return "http://localhost:8080/v1"
	}
}

func resolveAPIKey(id, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if strings.HasPrefix(id, "openai") {
		return os.Getenv("OPENAI_API_KEY")
	}
	return ""
}

func joinCompletionsPath(base string) string {
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}

type chatMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Reasoning string `json:"reasoning"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatCompletionResponse struct {
	Choices []chatChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}
