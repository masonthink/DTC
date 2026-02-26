// Package llm provides a unified gateway for all LLM API calls.
// It handles provider routing, cost tracking, caching, retry with
// exponential backoff, and failover between providers.
package llm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/digital-twin-community/backend/internal/config"
)

// Provider identifies an LLM provider.
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
	ProviderDeepSeek  Provider = "deepseek"
)

// TaskType categorizes the LLM call to select the right model.
type TaskType string

const (
	TaskDiscussionRound   TaskType = "discussion_round"   // 讨论发言生成
	TaskReportGeneration  TaskType = "report_generation"  // 报告正文生成
	TaskOpinionExtraction TaskType = "opinion_extraction" // 观点提取（轻量）
	TaskQualityEval       TaskType = "quality_eval"       // 质量自评（轻量）
	TaskTopicAnalysis     TaskType = "topic_analysis"     // Topic 分析
	TaskEmbedding         TaskType = "embedding"          // 向量化
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// Request is the unified LLM call request.
type Request struct {
	TaskType    TaskType
	Messages    []Message
	MaxTokens   int
	Temperature float32
	ReferenceID string // discussion_id / report_id for cost tracking
	Module      string // calling module name

	// 幂等键（相同 key 命中缓存返回相同结果）
	IdempotencyKey string
	// 是否允许从缓存读取
	UseCache bool
}

// Response is the unified LLM call response.
type Response struct {
	Content          string
	Provider         Provider
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          float64
	CachedHit        bool
	Duration         time.Duration
}

// DiscussionOutput is the structured output for discussion rounds.
type DiscussionOutput struct {
	Content     string  `json:"content"`
	KeyPoint    string  `json:"key_point"`
	AddressedTo string  `json:"addressed_to"`
	Confidence  float64 `json:"confidence"`
}

// OpinionMatrix is extracted from all discussion messages.
type OpinionMatrix struct {
	ConsensusPoints  []string `json:"consensus_points"`
	DivergencePoints []string `json:"divergence_points"`
	KeyQuestions     []string `json:"key_questions"`
	ActionItems      []string `json:"action_items"`
	BlindSpots       []string `json:"blind_spots"`
}

// Gateway is the central LLM call hub.
type Gateway struct {
	cfg         *config.LLMConfig
	redis       *redis.Client
	logger      *zap.Logger
	httpClient  *http.Client
	costTracker *CostTracker

	// circuit breaker state per provider
	failures map[Provider]*atomic.Int64
}

// NewGateway constructs a new Gateway.
func NewGateway(cfg *config.LLMConfig, rdb *redis.Client, logger *zap.Logger) *Gateway {
	failures := map[Provider]*atomic.Int64{
		ProviderAnthropic: new(atomic.Int64),
		ProviderOpenAI:    new(atomic.Int64),
		ProviderDeepSeek:  new(atomic.Int64),
	}
	return &Gateway{
		cfg:    cfg,
		redis:  rdb,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		costTracker: NewCostTracker(rdb, logger),
		failures:    failures,
	}
}

// Complete sends a chat completion request through the gateway.
// It handles caching, routing, retry, and failover automatically.
func (g *Gateway) Complete(ctx context.Context, req Request) (*Response, error) {
	// 1. 缓存检查
	if req.UseCache && req.IdempotencyKey != "" {
		if cached := g.getCache(ctx, req.IdempotencyKey); cached != nil {
			g.logger.Debug("llm cache hit", zap.String("key", req.IdempotencyKey))
			cached.CachedHit = true
			return cached, nil
		}
	}

	// 2. 选择 Provider 和 Model
	provider, model := g.selectProviderAndModel(req.TaskType)

	// 3. 执行调用（含重试和 Failover）
	resp, err := g.executeWithRetry(ctx, provider, model, req)
	if err != nil {
		// 尝试 Failover
		fallbackProvider, fallbackModel := g.fallbackProviderAndModel(provider, req.TaskType)
		g.logger.Warn("primary provider failed, trying fallback",
			zap.String("primary", string(provider)),
			zap.String("fallback", string(fallbackProvider)),
			zap.Error(err),
		)
		resp, err = g.executeWithRetry(ctx, fallbackProvider, fallbackModel, req)
		if err != nil {
			return nil, fmt.Errorf("all providers failed: %w", err)
		}
	}

	// 4. 成本追踪
	g.costTracker.Record(ctx, CostRecord{
		Provider:         resp.Provider,
		Model:            resp.Model,
		Module:           req.Module,
		ReferenceID:      req.ReferenceID,
		PromptTokens:     resp.PromptTokens,
		CompletionTokens: resp.CompletionTokens,
		TotalTokens:      resp.TotalTokens,
		CostUSD:          resp.CostUSD,
		Duration:         resp.Duration,
	})

	// 5. 写入缓存
	if req.UseCache && req.IdempotencyKey != "" {
		ttl := g.cacheTTLForTask(req.TaskType)
		g.setCache(ctx, req.IdempotencyKey, resp, ttl)
	}

	return resp, nil
}

// CompleteDiscussionRound sends a discussion round request and parses the structured output.
// It retries with correction instructions if the output format is invalid.
func (g *Gateway) CompleteDiscussionRound(ctx context.Context, req Request) (*DiscussionOutput, error) {
	const maxParseAttempts = 3

	var lastErr error
	for attempt := 1; attempt <= maxParseAttempts; attempt++ {
		if attempt > 1 {
			// 追加纠错指令
			req.Messages = append(req.Messages, Message{
				Role: "user",
				Content: fmt.Sprintf(
					"Your previous response (attempt %d) was not valid JSON matching the required format. "+
						"Please respond ONLY with valid JSON matching exactly: "+
						`{"content":"...","key_point":"...","addressed_to":"questioner|supporter|supplementer|inquirer","confidence":0.0-1.0}`,
					attempt-1,
				),
			})
		}

		resp, err := g.Complete(ctx, req)
		if err != nil {
			lastErr = err
			continue
		}

		output, err := parseDiscussionOutput(resp.Content)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", attempt, err)
			g.logger.Warn("discussion output parse failed, retrying",
				zap.Int("attempt", attempt),
				zap.Error(err),
			)
			continue
		}
		return output, nil
	}
	return nil, fmt.Errorf("failed to get valid discussion output after %d attempts: %w", maxParseAttempts, lastErr)
}

// Embed generates an embedding vector for the given text.
func (g *Gateway) Embed(ctx context.Context, text string, module string) ([]float32, error) {
	cacheKey := "embed:" + hashString(text)

	// 检查缓存
	if g.redis != nil {
		if cached, err := g.redis.Get(ctx, cacheKey).Bytes(); err == nil {
			var vec []float32
			if err := json.Unmarshal(cached, &vec); err == nil {
				return vec, nil
			}
		}
	}

	vec, err := g.callEmbeddingAPI(ctx, text)
	if err != nil {
		return nil, err
	}

	// 缓存 embedding
	if g.redis != nil && len(vec) > 0 {
		if b, err := json.Marshal(vec); err == nil {
			g.redis.Set(ctx, cacheKey, b, g.cfg.EmbeddingCacheTTL)
		}
	}

	return vec, nil
}

// selectProviderAndModel chooses provider and model based on task type.
func (g *Gateway) selectProviderAndModel(taskType TaskType) (Provider, string) {
	switch taskType {
	case TaskDiscussionRound, TaskReportGeneration:
		// 核心任务：使用主力模型
		return Provider(g.cfg.PrimaryProvider), g.cfg.PrimaryModel
	case TaskOpinionExtraction, TaskQualityEval, TaskTopicAnalysis:
		// 轻量任务：使用廉价模型节省成本
		return Provider(g.cfg.PrimaryProvider), g.cfg.LightModel
	default:
		return Provider(g.cfg.PrimaryProvider), g.cfg.PrimaryModel
	}
}

// fallbackProviderAndModel returns the fallback provider/model when primary fails.
func (g *Gateway) fallbackProviderAndModel(failed Provider, taskType TaskType) (Provider, string) {
	fallback := Provider(g.cfg.FallbackProvider)
	if fallback == failed {
		// 如果 fallback 和 failed 相同，尝试 OpenAI
		fallback = ProviderOpenAI
	}
	model := "deepseek-chat"
	if fallback == ProviderOpenAI {
		model = "gpt-4o-mini"
	}
	return fallback, model
}

// executeWithRetry calls the LLM with exponential backoff retry.
func (g *Gateway) executeWithRetry(ctx context.Context, provider Provider, model string, req Request) (*Response, error) {
	backoffDelays := []time.Duration{30 * time.Second, 60 * time.Second, 120 * time.Second}

	var lastErr error
	for attempt, delay := range append([]time.Duration{0}, backoffDelays...) {
		if delay > 0 {
			g.logger.Info("retrying llm call",
				zap.Int("attempt", attempt),
				zap.Duration("delay", delay),
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := g.callProvider(ctx, provider, model, req)
		if err == nil {
			g.failures[provider].Store(0) // reset failure count
			return resp, nil
		}

		lastErr = err
		g.failures[provider].Add(1)

		// 429 限流：继续重试
		// 其他错误：立即返回
		if !isRateLimitError(err) && !isTransientError(err) {
			return nil, err
		}

		g.logger.Warn("llm call failed, will retry",
			zap.String("provider", string(provider)),
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}
	return nil, fmt.Errorf("exhausted retries for provider %s: %w", provider, lastErr)
}

// callProvider dispatches to the appropriate provider implementation.
func (g *Gateway) callProvider(ctx context.Context, provider Provider, model string, req Request) (*Response, error) {
	start := time.Now()

	switch provider {
	case ProviderAnthropic:
		return g.callAnthropic(ctx, model, req, start)
	case ProviderDeepSeek:
		return g.callDeepSeek(ctx, model, req, start)
	case ProviderOpenAI:
		return g.callOpenAI(ctx, model, req, start)
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}

// callAnthropic calls the Anthropic Claude API.
func (g *Gateway) callAnthropic(ctx context.Context, model string, req Request, start time.Time) (*Response, error) {
	type anthropicMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type anthropicRequest struct {
		Model     string             `json:"model"`
		MaxTokens int                `json:"max_tokens"`
		System    string             `json:"system,omitempty"`
		Messages  []anthropicMessage `json:"messages"`
	}
	type anthropicUsage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	}
	type anthropicContent struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type anthropicResponse struct {
		ID      string             `json:"id"`
		Content []anthropicContent `json:"content"`
		Usage   anthropicUsage     `json:"usage"`
		Error   *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	// 分离 system message
	var systemMsg string
	var userMsgs []anthropicMessage
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemMsg = m.Content
		} else {
			userMsgs = append(userMsgs, anthropicMessage{Role: m.Role, Content: m.Content})
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	body := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    systemMsg,
		Messages:  userMsgs,
	}

	respData, statusCode, err := g.doHTTPRequest(ctx, "POST",
		"https://api.anthropic.com/v1/messages",
		map[string]string{
			"x-api-key":         g.cfg.AnthropicAPIKey,
			"anthropic-version": "2023-06-01",
		},
		body,
	)
	if err != nil {
		return nil, err
	}
	if statusCode == 429 {
		return nil, &RateLimitError{Provider: ProviderAnthropic}
	}
	if statusCode >= 500 {
		return nil, &TransientError{StatusCode: statusCode}
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respData, &apiResp); err != nil {
		return nil, fmt.Errorf("anthropic: decode response: %w", err)
	}
	if apiResp.Error != nil {
		return nil, fmt.Errorf("anthropic API error: %s: %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	var content string
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			content = c.Text
			break
		}
	}

	prompt := apiResp.Usage.InputTokens
	completion := apiResp.Usage.OutputTokens
	cost := calculateAnthropicCost(model, prompt, completion)

	return &Response{
		Content:          content,
		Provider:         ProviderAnthropic,
		Model:            model,
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      prompt + completion,
		CostUSD:          cost,
		Duration:         time.Since(start),
	}, nil
}

// callDeepSeek calls the DeepSeek API (OpenAI-compatible).
func (g *Gateway) callDeepSeek(ctx context.Context, model string, req Request, start time.Time) (*Response, error) {
	return g.callOpenAICompatible(ctx, "https://api.deepseek.com/v1/chat/completions",
		g.cfg.DeepSeekAPIKey, model, req, start, ProviderDeepSeek)
}

// callOpenAI calls the OpenAI API.
func (g *Gateway) callOpenAI(ctx context.Context, model string, req Request, start time.Time) (*Response, error) {
	return g.callOpenAICompatible(ctx, "https://api.openai.com/v1/chat/completions",
		g.cfg.OpenAIAPIKey, model, req, start, ProviderOpenAI)
}

// callOpenAICompatible calls any OpenAI-compatible API endpoint.
func (g *Gateway) callOpenAICompatible(ctx context.Context, endpoint, apiKey, model string, req Request, start time.Time, provider Provider) (*Response, error) {
	type chatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type chatRequest struct {
		Model       string        `json:"model"`
		Messages    []chatMessage `json:"messages"`
		MaxTokens   int           `json:"max_tokens,omitempty"`
		Temperature float32       `json:"temperature,omitempty"`
	}
	type chatUsage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	}
	type chatChoice struct {
		Message chatMessage `json:"message"`
	}
	type chatResponse struct {
		Choices []chatChoice `json:"choices"`
		Usage   chatUsage    `json:"usage"`
		Error   *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error,omitempty"`
	}

	msgs := make([]chatMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = chatMessage{Role: m.Role, Content: m.Content}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	body := chatRequest{
		Model:       model,
		Messages:    msgs,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
	}

	respData, statusCode, err := g.doHTTPRequest(ctx, "POST", endpoint,
		map[string]string{"Authorization": "Bearer " + apiKey},
		body,
	)
	if err != nil {
		return nil, err
	}
	if statusCode == 429 {
		return nil, &RateLimitError{Provider: provider}
	}
	if statusCode >= 500 {
		return nil, &TransientError{StatusCode: statusCode}
	}

	var apiResp chatResponse
	if err := json.Unmarshal(respData, &apiResp); err != nil {
		return nil, fmt.Errorf("%s: decode response: %w", provider, err)
	}
	if apiResp.Error != nil {
		return nil, fmt.Errorf("%s API error: %s", provider, apiResp.Error.Message)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("%s: empty choices in response", provider)
	}

	content := apiResp.Choices[0].Message.Content
	cost := calculateOpenAICost(model, apiResp.Usage.PromptTokens, apiResp.Usage.CompletionTokens)

	return &Response{
		Content:          content,
		Provider:         provider,
		Model:            model,
		PromptTokens:     apiResp.Usage.PromptTokens,
		CompletionTokens: apiResp.Usage.CompletionTokens,
		TotalTokens:      apiResp.Usage.TotalTokens,
		CostUSD:          cost,
		Duration:         time.Since(start),
	}, nil
}

// callEmbeddingAPI calls the Voyage embedding API.
func (g *Gateway) callEmbeddingAPI(ctx context.Context, text string) ([]float32, error) {
	type voyageRequest struct {
		Input          []string `json:"input"`
		Model          string   `json:"model"`
		InputType      string   `json:"input_type"`
	}
	type voyageEmbedding struct {
		Embedding []float32 `json:"embedding"`
	}
	type voyageResponse struct {
		Data []voyageEmbedding `json:"data"`
	}

	body := voyageRequest{
		Input:     []string{text},
		Model:     g.cfg.EmbeddingModel,
		InputType: "document",
	}

	respData, statusCode, err := g.doHTTPRequest(ctx, "POST",
		"https://api.voyageai.com/v1/embeddings",
		map[string]string{"Authorization": "Bearer " + g.cfg.VoyageAPIKey},
		body,
	)
	if err != nil {
		return nil, err
	}
	if statusCode != 200 {
		return nil, fmt.Errorf("voyage API returned status %d", statusCode)
	}

	var apiResp voyageResponse
	if err := json.Unmarshal(respData, &apiResp); err != nil {
		return nil, fmt.Errorf("voyage: decode response: %w", err)
	}
	if len(apiResp.Data) == 0 {
		return nil, fmt.Errorf("voyage: empty embedding response")
	}
	return apiResp.Data[0].Embedding, nil
}

// doHTTPRequest performs an authenticated HTTP request.
func (g *Gateway) doHTTPRequest(ctx context.Context, method, url string, headers map[string]string, body interface{}) ([]byte, int, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(string(b)))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	var buf []byte
	buf = make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if readErr != nil {
			break
		}
	}
	return buf, resp.StatusCode, nil
}

// Cache helpers

func (g *Gateway) cacheKey(idempotencyKey string) string {
	return "llm:cache:" + hashString(idempotencyKey)
}

func (g *Gateway) getCache(ctx context.Context, idempotencyKey string) *Response {
	if g.redis == nil {
		return nil
	}
	data, err := g.redis.Get(ctx, g.cacheKey(idempotencyKey)).Bytes()
	if err != nil {
		return nil
	}
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil
	}
	return &resp
}

func (g *Gateway) setCache(ctx context.Context, idempotencyKey string, resp *Response, ttl time.Duration) {
	if g.redis == nil {
		return
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	g.redis.Set(ctx, g.cacheKey(idempotencyKey), data, ttl)
}

func (g *Gateway) cacheTTLForTask(taskType TaskType) time.Duration {
	switch taskType {
	case TaskEmbedding:
		return g.cfg.EmbeddingCacheTTL
	case TaskReportGeneration:
		return g.cfg.ReportCacheTTL
	default:
		return g.cfg.PromptCacheTTL
	}
}

// Cost calculation

func calculateAnthropicCost(model string, promptTokens, completionTokens int) float64 {
	// Pricing per 1M tokens (approximate)
	prices := map[string][2]float64{
		"claude-sonnet-4-6":          {3.0, 15.0},
		"claude-opus-4-6":            {15.0, 75.0},
		"claude-haiku-4-5-20251001": {0.25, 1.25},
	}
	p, ok := prices[model]
	if !ok {
		p = [2]float64{3.0, 15.0}
	}
	return float64(promptTokens)/1_000_000*p[0] + float64(completionTokens)/1_000_000*p[1]
}

func calculateOpenAICost(model string, promptTokens, completionTokens int) float64 {
	prices := map[string][2]float64{
		"gpt-4o":      {5.0, 15.0},
		"gpt-4o-mini": {0.15, 0.60},
		"deepseek-chat": {0.14, 0.28},
	}
	p, ok := prices[model]
	if !ok {
		p = [2]float64{0.14, 0.28}
	}
	return float64(promptTokens)/1_000_000*p[0] + float64(completionTokens)/1_000_000*p[1]
}

// parseDiscussionOutput parses the LLM JSON response for discussion rounds.
func parseDiscussionOutput(content string) (*DiscussionOutput, error) {
	// 尝试从 content 中提取 JSON（可能包裹在 markdown code block 中）
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) >= 3 {
			content = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var output DiscussionOutput
	if err := json.Unmarshal([]byte(content), &output); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if output.Content == "" {
		return nil, fmt.Errorf("missing required field: content")
	}
	if output.KeyPoint == "" {
		return nil, fmt.Errorf("missing required field: key_point")
	}
	// Clamp confidence
	if output.Confidence < 0 {
		output.Confidence = 0
	}
	if output.Confidence > 1 {
		output.Confidence = 1
	}
	return &output, nil
}

// Error types

type RateLimitError struct {
	Provider Provider
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded for provider %s", e.Provider)
}

type TransientError struct {
	StatusCode int
}

func (e *TransientError) Error() string {
	return fmt.Sprintf("transient error: HTTP %d", e.StatusCode)
}

func isRateLimitError(err error) bool {
	_, ok := err.(*RateLimitError)
	return ok
}

func isTransientError(err error) bool {
	_, ok := err.(*TransientError)
	return ok
}

// utilities

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:8])
}

// exponentialBackoff returns the delay for the nth retry.
func exponentialBackoff(attempt int, base time.Duration) time.Duration {
	delay := float64(base) * math.Pow(2, float64(attempt))
	max := float64(120 * time.Second)
	if delay > max {
		delay = max
	}
	return time.Duration(delay)
}

// =============================================================================
// CostTracker
// =============================================================================

// CostRecord captures cost information for a single LLM call.
type CostRecord struct {
	Provider         Provider
	Model            string
	Module           string
	ReferenceID      string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          float64
	Duration         time.Duration
}

// CostTracker records LLM usage and cost to Redis for real-time dashboards.
type CostTracker struct {
	redis  *redis.Client
	logger *zap.Logger
}

// NewCostTracker constructs a CostTracker.
func NewCostTracker(rdb *redis.Client, logger *zap.Logger) *CostTracker {
	return &CostTracker{redis: rdb, logger: logger}
}

// Record asynchronously records cost to Redis.
// Database persistence is handled by a background flush goroutine.
func (ct *CostTracker) Record(ctx context.Context, record CostRecord) {
	if ct.redis == nil {
		return
	}
	// 实时累计当日成本（用于告警）
	dayKey := fmt.Sprintf("llm:cost:day:%s", time.Now().Format("2006-01-02"))
	ct.redis.IncrByFloat(ctx, dayKey, record.CostUSD)
	ct.redis.Expire(ctx, dayKey, 48*time.Hour)

	// 模块级别统计
	moduleKey := fmt.Sprintf("llm:cost:module:%s:%s", record.Module, time.Now().Format("2006-01-02"))
	ct.redis.IncrByFloat(ctx, moduleKey, record.CostUSD)
	ct.redis.Expire(ctx, moduleKey, 7*24*time.Hour)
}

// GetDailyCost returns the total LLM cost for today.
func (ct *CostTracker) GetDailyCost(ctx context.Context) float64 {
	dayKey := fmt.Sprintf("llm:cost:day:%s", time.Now().Format("2006-01-02"))
	val, err := ct.redis.Get(ctx, dayKey).Float64()
	if err != nil {
		return 0
	}
	return val
}
