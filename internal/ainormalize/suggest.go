package ainormalize

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"one-cli/internal/configgen"
	"one-cli/internal/openapi"

	"gopkg.in/yaml.v3"
)

type Inventory struct {
	Title      string               `json:"title"`
	Operations []InventoryOperation `json:"operations"`
}

type InventoryOperation struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Tag         string `json:"tag"`
	OperationID string `json:"operationId,omitempty"`
	Summary     string `json:"summary,omitempty"`
}

type Suggestion struct {
	TagAlias       map[string]string `json:"tag_alias"`
	OperationAlias map[string]string `json:"operation_alias"`
}

type Diagnostics struct {
	Rejected []RejectedSuggestion
}

type RejectedSuggestion struct {
	Kind   string
	Key    string
	Alias  string
	Reason string
}

type Client interface {
	SuggestConfig(context.Context, Inventory) (Suggestion, error)
}

type ClientFunc func(context.Context, Inventory) (Suggestion, error)

func (fn ClientFunc) SuggestConfig(ctx context.Context, inventory Inventory) (Suggestion, error) {
	return fn(ctx, inventory)
}

func BuildInventory(doc openapi.Document) Inventory {
	operations := make([]InventoryOperation, 0, len(doc.Operations))
	for _, op := range doc.Operations {
		operations = append(operations, InventoryOperation{
			Method:      strings.ToUpper(strings.TrimSpace(op.Method)),
			Path:        strings.TrimSpace(op.Path),
			Tag:         strings.TrimSpace(op.Tag),
			OperationID: strings.TrimSpace(op.OperationID),
			Summary:     strings.TrimSpace(op.Summary),
		})
	}
	return Inventory{
		Title:      strings.TrimSpace(doc.Title),
		Operations: operations,
	}
}

func ValidateSuggestion(inventory Inventory, suggestion Suggestion) (configgen.Config, Diagnostics) {
	var diagnostics Diagnostics
	cfg := configgen.Config{
		Naming: configgen.NamingConfig{
			TagAlias:       map[string]string{},
			OperationAlias: map[string]string{},
		},
	}

	knownTags := map[string]struct{}{}
	operationsByKey := map[string][]InventoryOperation{}
	for _, op := range inventory.Operations {
		if op.Tag != "" {
			knownTags[op.Tag] = struct{}{}
		}
		for _, key := range operationKeys(op) {
			operationsByKey[key] = append(operationsByKey[key], op)
		}
	}

	for _, key := range sortedKeys(suggestion.TagAlias) {
		alias := strings.TrimSpace(suggestion.TagAlias[key])
		if _, ok := knownTags[key]; !ok {
			diagnostics.reject("tag_alias", key, alias, "unknown tag")
			continue
		}
		if !isCLIName(alias) {
			diagnostics.reject("tag_alias", key, alias, "alias must be lowercase ASCII letters, numbers, and hyphens")
			continue
		}
		cfg.Naming.TagAlias[key] = alias
	}

	seenByGroup := map[string]map[string]string{}
	for _, key := range sortedKeys(suggestion.OperationAlias) {
		alias := strings.TrimSpace(suggestion.OperationAlias[key])
		matches, ok := operationsByKey[key]
		if !ok {
			diagnostics.reject("operation_alias", key, alias, "unknown operation")
			continue
		}
		if !isCLIName(alias) {
			diagnostics.reject("operation_alias", key, alias, "alias must be lowercase ASCII letters, numbers, and hyphens")
			continue
		}
		var rejected bool
		for _, op := range matches {
			group := cfg.Naming.TagAlias[op.Tag]
			if group == "" {
				group = op.Tag
			}
			if seenByGroup[group] == nil {
				seenByGroup[group] = map[string]string{}
			}
			if existing := seenByGroup[group][alias]; existing != "" {
				diagnostics.reject("operation_alias", key, alias, fmt.Sprintf("duplicate alias in group %q also used by %s", group, existing))
				rejected = true
				break
			}
		}
		if rejected {
			continue
		}
		for _, op := range matches {
			group := cfg.Naming.TagAlias[op.Tag]
			if group == "" {
				group = op.Tag
			}
			seenByGroup[group][alias] = key
		}
		cfg.Naming.OperationAlias[key] = alias
	}

	if len(cfg.Naming.TagAlias) == 0 {
		cfg.Naming.TagAlias = nil
	}
	if len(cfg.Naming.OperationAlias) == 0 {
		cfg.Naming.OperationAlias = nil
	}
	return cfg, diagnostics
}

func RenderConfigYAML(cfg configgen.Config) ([]byte, error) {
	return yaml.Marshal(struct {
		Naming configgen.NamingConfig `yaml:"naming"`
	}{
		Naming: cfg.Naming,
	})
}

func NewOpenAICompatibleClientFromEnv() (Client, error) {
	baseURL := strings.TrimSpace(os.Getenv("OPENCLI_AI_BASE_URL"))
	apiKey := strings.TrimSpace(os.Getenv("OPENCLI_AI_API_KEY"))
	model := strings.TrimSpace(os.Getenv("OPENCLI_AI_MODEL"))
	switch {
	case baseURL == "":
		return nil, fmt.Errorf("OPENCLI_AI_BASE_URL is required")
	case apiKey == "":
		return nil, fmt.Errorf("OPENCLI_AI_API_KEY is required")
	case model == "":
		return nil, fmt.Errorf("OPENCLI_AI_MODEL is required")
	}
	return OpenAICompatibleClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		Client:  &http.Client{Timeout: 60 * time.Second},
	}, nil
}

type OpenAICompatibleClient struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

func (client OpenAICompatibleClient) SuggestConfig(ctx context.Context, inventory Inventory) (Suggestion, error) {
	httpClient := client.Client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	endpoint, err := chatCompletionsEndpoint(client.BaseURL)
	if err != nil {
		return Suggestion{}, err
	}
	body := chatCompletionsRequest{
		Model:       client.Model,
		Temperature: 0,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: inventoryPrompt(inventory)},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return Suggestion{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return Suggestion{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.APIKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return Suggestion{}, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Suggestion{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Suggestion{}, fmt.Errorf("AI provider returned HTTP %d: %s", resp.StatusCode, sanitizeRespBody(respBody))
	}
	var parsed chatCompletionsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Suggestion{}, fmt.Errorf("decode AI response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return Suggestion{}, fmt.Errorf("AI response did not include choices")
	}
	content := stripCodeFences(strings.TrimSpace(parsed.Choices[0].Message.Content))
	var suggestion Suggestion
	if err := json.Unmarshal([]byte(content), &suggestion); err != nil {
		preview := content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return Suggestion{}, fmt.Errorf("decode AI suggestion JSON: %w (content preview: %s)", err, preview)
	}
	return suggestion, nil
}

func (diagnostics *Diagnostics) reject(kind, key, alias, reason string) {
	diagnostics.Rejected = append(diagnostics.Rejected, RejectedSuggestion{
		Kind:   kind,
		Key:    key,
		Alias:  alias,
		Reason: reason,
	})
}

func operationKeys(op InventoryOperation) []string {
	var keys []string
	if op.OperationID != "" {
		keys = append(keys, op.OperationID)
	}
	if op.Method != "" && op.Path != "" {
		keys = append(keys, op.Method+" "+op.Path)
	}
	if op.Path != "" {
		keys = append(keys, op.Path)
	}
	return keys
}

func isCLIName(value string) bool {
	if value == "" || strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
		return false
	}
	lastHyphen := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			lastHyphen = false
		case r == '-':
			if lastHyphen {
				return false
			}
			lastHyphen = true
		default:
			return false
		}
	}
	return true
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func chatCompletionsEndpoint(base string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("OPENCLI_AI_BASE_URL must be an absolute URL")
	}
	path := strings.TrimRight(parsed.Path, "/")
	switch {
	case strings.HasSuffix(path, "/chat/completions"):
		parsed.Path = path
	case strings.HasSuffix(path, "/v1"):
		parsed.Path = path + "/chat/completions"
	default:
		parsed.Path = path + "/v1/chat/completions"
	}
	return parsed.String(), nil
}

func inventoryPrompt(inventory Inventory) string {
	raw, err := json.Marshal(inventory)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func stripCodeFences(content string) string {
	content = strings.TrimSpace(content)
	for _, fence := range []string{"```json", "```JSON", "```Json", "```"} {
		if strings.HasPrefix(content, fence) {
			content = strings.TrimSpace(strings.TrimPrefix(content, fence))
			break
		}
	}
	if strings.HasSuffix(content, "```") {
		content = strings.TrimSpace(strings.TrimSuffix(content, "```"))
	}
	return strings.TrimSpace(content)
}

func sanitizeRespBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	for _, marker := range []string{"\"api-key\"", "\"apiKey\"", "authorization", "Authorization", "bearer"} {
		if idx := strings.Index(strings.ToLower(text), strings.ToLower(marker)); idx >= 0 {
			text = text[:idx] + "[redacted]"
			break
		}
	}
	const max = 500
	if len(text) > max {
		text = text[:max] + "..."
	}
	return text
}

const systemPrompt = `You generate reviewable opencli.yaml naming suggestions for irregular OpenAPI documents.
Return only JSON with this shape: {"tag_alias":{"original tag":"cli-group"},"operation_alias":{"METHOD /path":"cli-command"}}.
Prefer METHOD /path operation_alias keys because they are stable when operationId is missing, duplicated, or generated by an internal platform.
Use concise lowercase English identifiers with ASCII letters, numbers, and hyphens. Keep command aliases to 2-3 words when possible.
Map Chinese, Japanese, Korean, special-character, or keyword-like tags to short business domain group names such as logistics, supplier, order, catalog, or user.
Prefer business meaning from summary over raw path tokens. If summary is empty, infer only from method, path, tag, and operationId.
Remove transport and vendor noise such as api, api-apply, version segments, v1, v2, supplier, get, query, list, push, controller, usingpost, and usingget when redundant.
Keep verbs only when they clarify the user action or risk, such as confirm-po, sync-inventory, submit-order, delete-user, or approve-request.
Do not infer schemas, request fields, response fields, signing, securitySchemes, servers, or body_mode. Do not infer auth. This feature only suggests naming aliases.
Do not rewrite paths, methods, operationId values, or OpenAPI content.`

type chatCompletionsRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionsResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}
