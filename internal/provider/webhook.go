package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bkelceoglu/go-rt/internal/domain"
)

var _ domain.Provider = (*WebhookProvider)(nil)

// WebhookProvider executes arbitrary REST HTTP endpoints, API Gateways, and custom chatbot services.
type WebhookProvider struct {
	spec       domain.ProviderSpec
	targetURL  string
	httpClient *http.Client
}

// NewWebhook creates a configured WebhookProvider from ProviderSpec.
func NewWebhook(spec domain.ProviderSpec, client *http.Client) *WebhookProvider {
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	return &WebhookProvider{
		spec:       spec,
		targetURL:  resolveWebhookURL(spec.ID, spec.BaseURL),
		httpClient: client,
	}
}

// ID returns the webhook identifier.
func (w *WebhookProvider) ID() string {
	return w.spec.ID
}

// Call executes the HTTP request against the configured webhook/gateway.
func (w *WebhookProvider) Call(ctx context.Context, req domain.PromptRequest) (domain.ProviderResponse, error) {
	httpReq, err := w.prepareWebhookRequest(ctx, req)
	if err != nil {
		return domain.ProviderResponse{}, err
	}
	start := time.Now()
	respBytes, statusCode, err := w.executeHTTP(httpReq)
	if err != nil {
		return domain.ProviderResponse{}, err
	}
	return w.parseResponse(respBytes, statusCode, time.Since(start).Milliseconds())
}

func (w *WebhookProvider) prepareWebhookRequest(ctx context.Context, req domain.PromptRequest) (*http.Request, error) {
	bodyBytes, err := w.renderBody(req)
	if err != nil {
		return nil, err
	}
	return w.buildHTTPRequest(ctx, bodyBytes)
}

func (w *WebhookProvider) renderBody(req domain.PromptRequest) ([]byte, error) {
	if w.spec.BodyTmpl != "" {
		return renderBodyTemplate(w.spec.BodyTmpl, req)
	}
	defaultPayload := map[string]string{"prompt": req.Prompt}
	data, err := json.Marshal(defaultPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal webhook payload: %w", err)
	}
	return data, nil
}

func renderBodyTemplate(tmpl string, req domain.PromptRequest) ([]byte, error) {
	cleanPrompt, err := escapeJSONString(req.Prompt)
	if err != nil {
		return nil, err
	}
	rendered := strings.ReplaceAll(tmpl, "{{prompt}}", cleanPrompt)
	for k, v := range req.Vars {
		cleanVal, err := escapeJSONString(v)
		if err != nil {
			return nil, err
		}
		rendered = strings.ReplaceAll(rendered, "{{"+k+"}}", cleanVal)
	}
	return []byte(rendered), nil
}

func escapeJSONString(val string) (string, error) {
	escaped, err := json.Marshal(val)
	if err != nil {
		return "", fmt.Errorf("marshal template string: %w", err)
	}
	return strings.Trim(string(escaped), `"`), nil
}

func (w *WebhookProvider) buildHTTPRequest(ctx context.Context, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create webhook request for %s: %w", w.spec.ID, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if w.spec.APIKey != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+w.spec.APIKey)
	}
	for k, v := range w.spec.Headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

func (w *WebhookProvider) executeHTTP(req *http.Request) ([]byte, int, error) {
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute webhook %s: %w", w.spec.ID, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response from %s: %w", w.spec.ID, err)
	}
	return data, resp.StatusCode, nil
}

func (w *WebhookProvider) parseResponse(data []byte, status int, latency int64) (domain.ProviderResponse, error) {
	if status < 200 || status >= 300 {
		return domain.ProviderResponse{}, fmt.Errorf("webhook %s returned http %d: %s", w.spec.ID, status, string(data))
	}
	output, err := extractResponseText(data, w.spec.RespPath)
	if err != nil {
		return domain.ProviderResponse{}, err
	}
	return domain.ProviderResponse{
		Output:     output,
		LatencyMs:  latency,
		TokenUsage: domain.TokenUsage{Requests: 1},
	}, nil
}

func extractResponseText(data []byte, respPath string) (string, error) {
	if respPath == "" {
		return fallbackExtract(data), nil
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return string(data), nil
	}
	val, err := traverseJSONPath(root, strings.Split(respPath, "."))
	if err != nil {
		return "", err
	}
	return fmt.Sprint(val), nil
}

func traverseJSONPath(curr any, parts []string) (any, error) {
	if len(parts) == 0 {
		return curr, nil
	}
	next, err := stepJSONPath(curr, parts[0])
	if err != nil {
		return nil, err
	}
	return traverseJSONPath(next, parts[1:])
}

func stepJSONPath(curr any, key string) (any, error) {
	switch v := curr.(type) {
	case map[string]any:
		child, ok := v[key]
		if !ok {
			return nil, fmt.Errorf("key %q not found in response json", key)
		}
		return child, nil
	case []any:
		return stepJSONSlice(v, key)
	default:
		return nil, fmt.Errorf("cannot traverse into %T for key %q", curr, key)
	}
}

func stepJSONSlice(slice []any, key string) (any, error) {
	idx, err := strconv.Atoi(key)
	if err != nil || idx < 0 || idx >= len(slice) {
		return nil, fmt.Errorf("invalid array index %q in response json", key)
	}
	return slice[idx], nil
}

func fallbackExtract(data []byte) string {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return string(data)
	}
	candidates := []string{"output", "reply", "response", "message", "text", "content"}
	for _, c := range candidates {
		if val, ok := m[c]; ok {
			return fmt.Sprint(val)
		}
	}
	return string(data)
}

func resolveWebhookURL(id, explicit string) string {
	if explicit != "" {
		return explicit
	}
	clean := strings.TrimPrefix(id, "webhook:")
	if strings.HasPrefix(clean, "http://") || strings.HasPrefix(clean, "https://") {
		return clean
	}
	return "http://" + clean
}
