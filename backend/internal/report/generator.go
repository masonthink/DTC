// Package report implements the 4-step report generation pipeline.
//
// Pipeline:
//   Step 1: Aggregate key_points from all 16 messages (4 roles × 4 rounds)
//   Step 2: Extract opinion matrix (consensus, divergence, questions, actions, blind_spots)
//   Step 3: Generate 800-1200 word report from opinion matrix (not raw messages)
//   Step 4: Quality evaluation (LLM self-score < 7 → rewrite) + rule checks
package report

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"

	"github.com/digital-twin-community/backend/internal/llm"
)

// DiscussionMessage represents a single message from the discussion.
type DiscussionMessage struct {
	DiscussionID string
	AgentID      string
	AnonID       string
	RoundNumber  int
	Role         string // questioner, supporter, supplementer, inquirer
	Content      string
	KeyPoint     string // 核心观点摘要（Step 1 的输入）
	Confidence   float64
	ModelUsed    string
}

// OpinionMatrix is extracted in Step 2.
type OpinionMatrix struct {
	ConsensusPoints  []string `json:"consensus_points"`
	DivergencePoints []string `json:"divergence_points"`
	KeyQuestions     []string `json:"key_questions"`
	ActionItems      []string `json:"action_items"`
	BlindSpots       []string `json:"blind_spots"`
}

// ConnectionCandidate is scored for recommendation.
type ConnectionCandidate struct {
	AgentID     string
	AnonID      string
	InsightCount    int     // 被引用次数 + 洞见数量 + 改变讨论走向次数
	Complementarity float64 // 与提交者背景差异度（适中最优，0-1）
	CollabSignal    float64 // 历史被申请频次 + 接受率（0-1）
	ActivityScore   float64 // 最近活跃 + 历史响应速度（0-1）
}

// RecommendedAgent is an agent recommended for connection.
type RecommendedAgent struct {
	AgentID     string  `json:"agent_id"`
	AnonID      string  `json:"anon_id"`
	FinalScore  float64 `json:"final_score"`
	ScoreBreakdown ScoreBreakdown `json:"score_breakdown"`
	Reasons     []string `json:"reasons"`
}

// ScoreBreakdown shows how the connection score was computed.
type ScoreBreakdown struct {
	InsightScore       float64 `json:"insight_score"`        // 40%
	ComplementaryScore float64 `json:"complementary_score"`  // 30%
	CollabScore        float64 `json:"collab_score"`         // 20%
	ActivityScore      float64 `json:"activity_score"`       // 10%
}

// Report is the final generated report.
type Report struct {
	ID              string        // database primary key
	DiscussionID    string
	TopicID         string
	Summary         string        // 800-1200 字报告正文
	OpinionMatrix   OpinionMatrix
	RecommendedAgents []RecommendedAgent
	QualityScore    float64       // LLM 自评 1-10
	UserRating      int           // 1-5 user-provided rating
	UserFeedback    string
	ModelUsed       string
	TotalTokens     int
	TotalCostUSD    float64
	GenerationAttempts int
	GeneratedAt     time.Time
}

// Topic holds the topic context for report generation.
type Topic struct {
	ID          string
	Title       string
	Description string
	TopicType   string
	Tags        []string
}

// GeneratorConfig holds tuning parameters.
type GeneratorConfig struct {
	MinWords          int     // 最小字数（默认 800）
	MaxWords          int     // 最大字数（默认 1200）
	MinQualityScore   float64 // 低于此分重写（默认 7.0）
	MaxRewriteAttempts int    // 最大重写次数（默认 2）
	MaxRecommendations int   // 最多推荐几个 Agent（默认 3）
}

// DefaultConfig returns production-ready defaults.
func DefaultConfig() GeneratorConfig {
	return GeneratorConfig{
		MinWords:           800,
		MaxWords:           1200,
		MinQualityScore:    7.0,
		MaxRewriteAttempts: 2,
		MaxRecommendations: 3,
	}
}

// Generator runs the 4-step report pipeline.
type Generator struct {
	llm    *llm.Gateway
	logger *zap.Logger
	cfg    GeneratorConfig
}

// NewGenerator constructs a Generator.
func NewGenerator(gateway *llm.Gateway, logger *zap.Logger, cfg GeneratorConfig) *Generator {
	return &Generator{llm: gateway, logger: logger, cfg: cfg}
}

// Generate runs the full 4-step pipeline and returns a Report.
func (g *Generator) Generate(
	ctx context.Context,
	topic Topic,
	messages []DiscussionMessage,
	candidates []ConnectionCandidate,
) (*Report, error) {
	g.logger.Info("starting report generation",
		zap.String("discussion_id", messages[0].DiscussionID),
		zap.Int("message_count", len(messages)),
	)

	report := &Report{
		DiscussionID: messages[0].DiscussionID,
		TopicID:      topic.ID,
		GeneratedAt:  time.Now(),
	}

	// Step 1: Aggregate key_points
	keyPoints := g.aggregateKeyPoints(messages)
	g.logger.Debug("step 1 complete: key points aggregated", zap.Int("count", len(keyPoints)))

	// Step 2: Extract opinion matrix
	matrix, tokens2, cost2, err := g.extractOpinionMatrix(ctx, topic, keyPoints, messages[0].DiscussionID)
	if err != nil {
		return nil, fmt.Errorf("step 2 (opinion matrix): %w", err)
	}
	report.OpinionMatrix = *matrix
	report.TotalTokens += tokens2
	report.TotalCostUSD += cost2
	g.logger.Debug("step 2 complete: opinion matrix extracted")

	// Step 3: Generate report summary
	summary, model, tokens3, cost3, err := g.generateSummary(ctx, topic, matrix, messages[0].DiscussionID)
	if err != nil {
		return nil, fmt.Errorf("step 3 (summary): %w", err)
	}
	report.Summary = summary
	report.ModelUsed = model
	report.TotalTokens += tokens3
	report.TotalCostUSD += cost3
	g.logger.Debug("step 3 complete: summary generated", zap.Int("word_count", countChineseWords(summary)))

	// Step 4: Quality evaluation + auto-rewrite
	qualityScore, rewriteAttempts, extraTokens, extraCost, err := g.evaluateAndImprove(ctx, topic, report, matrix)
	if err != nil {
		g.logger.Warn("quality evaluation failed, keeping original", zap.Error(err))
		qualityScore = 0
	}
	report.QualityScore = qualityScore
	report.GenerationAttempts = 1 + rewriteAttempts
	report.TotalTokens += extraTokens
	report.TotalCostUSD += extraCost

	// Step 5: Connection recommendations
	report.RecommendedAgents = g.scoreConnectionCandidates(candidates, messages)

	g.logger.Info("report generation complete",
		zap.String("discussion_id", report.DiscussionID),
		zap.Float64("quality_score", report.QualityScore),
		zap.Int("attempts", report.GenerationAttempts),
		zap.Int("total_tokens", report.TotalTokens),
	)

	return report, nil
}

// Step 1: Aggregate key_points from all messages, organized by role and round.
func (g *Generator) aggregateKeyPoints(messages []DiscussionMessage) []KeyPointEntry {
	entries := make([]KeyPointEntry, 0, len(messages))
	for _, msg := range messages {
		entries = append(entries, KeyPointEntry{
			Role:        msg.Role,
			AnonID:      msg.AnonID,
			RoundNumber: msg.RoundNumber,
			KeyPoint:    msg.KeyPoint,
			Confidence:  msg.Confidence,
		})
	}
	return entries
}

// KeyPointEntry is a single key_point with metadata.
type KeyPointEntry struct {
	Role        string
	AnonID      string
	RoundNumber int
	KeyPoint    string
	Confidence  float64
}

// Step 2: Extract opinion matrix using a light LLM.
func (g *Generator) extractOpinionMatrix(
	ctx context.Context,
	topic Topic,
	keyPoints []KeyPointEntry,
	discussionID string,
) (*OpinionMatrix, int, float64, error) {
	prompt := g.buildOpinionExtractionPrompt(topic, keyPoints)

	cacheKey := fmt.Sprintf("report:matrix:%s", discussionID)
	resp, err := g.llm.Complete(ctx, llm.Request{
		TaskType: llm.TaskOpinionExtraction,
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:      2048,
		Temperature:    0.3,
		Module:         "report",
		ReferenceID:    discussionID,
		IdempotencyKey: cacheKey,
		UseCache:       true,
	})
	if err != nil {
		return nil, 0, 0, err
	}

	matrix, err := parseOpinionMatrix(resp.Content)
	if err != nil {
		return nil, resp.TotalTokens, resp.CostUSD, fmt.Errorf("parse opinion matrix: %w", err)
	}
	return matrix, resp.TotalTokens, resp.CostUSD, nil
}

// Step 3: Generate report summary from opinion matrix (NOT raw discussion content).
func (g *Generator) generateSummary(
	ctx context.Context,
	topic Topic,
	matrix *OpinionMatrix,
	discussionID string,
) (string, string, int, float64, error) {
	prompt := g.buildSummaryPrompt(topic, matrix)

	cacheKey := fmt.Sprintf("report:summary:%s", discussionID)
	resp, err := g.llm.Complete(ctx, llm.Request{
		TaskType: llm.TaskReportGeneration,
		Messages: []llm.Message{
			{Role: "system", Content: g.reportSystemPrompt()},
			{Role: "user", Content: prompt},
		},
		MaxTokens:      2048,
		Temperature:    0.7,
		Module:         "report",
		ReferenceID:    discussionID,
		IdempotencyKey: cacheKey,
		UseCache:       true,
	})
	if err != nil {
		return "", "", 0, 0, err
	}

	return resp.Content, resp.Model, resp.TotalTokens, resp.CostUSD, nil
}

// Step 4: Quality evaluation and auto-rewrite.
func (g *Generator) evaluateAndImprove(
	ctx context.Context,
	topic Topic,
	report *Report,
	matrix *OpinionMatrix,
) (float64, int, int, float64, error) {
	var totalTokens int
	var totalCost float64
	var rewriteAttempts int

	currentSummary := report.Summary

	for attempt := 0; attempt <= g.cfg.MaxRewriteAttempts; attempt++ {
		// Rule-based checks first (fast, no LLM cost)
		ruleIssues := g.ruleCheck(currentSummary)

		// LLM quality self-evaluation
		score, evalTokens, evalCost, err := g.selfEvaluate(ctx, topic, currentSummary, matrix, report.DiscussionID)
		totalTokens += evalTokens
		totalCost += evalCost
		if err != nil {
			return 0, rewriteAttempts, totalTokens, totalCost, err
		}

		g.logger.Debug("quality evaluation",
			zap.Int("attempt", attempt),
			zap.Float64("score", score),
			zap.Strings("rule_issues", ruleIssues),
		)

		if score >= g.cfg.MinQualityScore && len(ruleIssues) == 0 {
			report.Summary = currentSummary
			return score, rewriteAttempts, totalTokens, totalCost, nil
		}

		if attempt == g.cfg.MaxRewriteAttempts {
			// 已达最大重写次数，使用当前版本
			report.Summary = currentSummary
			return score, rewriteAttempts, totalTokens, totalCost, nil
		}

		// 触发重写
		rewriteAttempts++
		g.logger.Info("rewriting report",
			zap.Float64("score", score),
			zap.Int("attempt", rewriteAttempts),
		)
		rewritten, rewriteTokens, rewriteCost, err := g.rewrite(ctx, topic, currentSummary, matrix, score, ruleIssues, report.DiscussionID)
		totalTokens += rewriteTokens
		totalCost += rewriteCost
		if err != nil {
			g.logger.Warn("rewrite failed, keeping current", zap.Error(err))
			return score, rewriteAttempts, totalTokens, totalCost, nil
		}
		currentSummary = rewritten
	}

	return 0, rewriteAttempts, totalTokens, totalCost, nil
}

// selfEvaluate asks the LLM to score the report quality (1-10).
func (g *Generator) selfEvaluate(
	ctx context.Context,
	topic Topic,
	summary string,
	matrix *OpinionMatrix,
	discussionID string,
) (float64, int, float64, error) {
	evalPrompt := fmt.Sprintf(`你是报告质量评估专家。请对以下讨论报告进行质量评分（1-10分）。

**评分维度：**
1. 内容深度（1-3分）：观点是否深刻，有无泛泛而谈
2. 观点平衡（1-3分）：是否呈现了共识与分歧，视角是否多元
3. 实用价值（1-2分）：行动建议是否具体可执行
4. 表达清晰（1-2分）：逻辑结构是否清晰，文字是否流畅

**原始观点矩阵（参考）：**
共识：%s
分歧：%s

**待评估报告：**
%s

请只返回一个JSON对象：{"score": <数字>, "feedback": "<简要反馈，不超过100字>"}`,
		strings.Join(matrix.ConsensusPoints, "；"),
		strings.Join(matrix.DivergencePoints, "；"),
		summary,
	)

	resp, err := g.llm.Complete(ctx, llm.Request{
		TaskType:    llm.TaskQualityEval,
		Messages:    []llm.Message{{Role: "user", Content: evalPrompt}},
		MaxTokens:   256,
		Temperature: 0.1,
		Module:      "report",
		ReferenceID: discussionID,
	})
	if err != nil {
		return 0, 0, 0, err
	}

	var result struct {
		Score    float64 `json:"score"`
		Feedback string  `json:"feedback"`
	}
	content := strings.TrimSpace(resp.Content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) >= 3 {
			content = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// 无法解析时返回中等分数，不阻塞流程
		return 6.0, resp.TotalTokens, resp.CostUSD, nil
	}
	return result.Score, resp.TotalTokens, resp.CostUSD, nil
}

// rewrite requests an improved version of the report.
func (g *Generator) rewrite(
	ctx context.Context,
	topic Topic,
	originalSummary string,
	matrix *OpinionMatrix,
	score float64,
	ruleIssues []string,
	discussionID string,
) (string, int, float64, error) {
	issuesStr := ""
	if len(ruleIssues) > 0 {
		issuesStr = "\n规则问题需修复：" + strings.Join(ruleIssues, "、")
	}

	rewritePrompt := fmt.Sprintf(`原始报告质量评分为 %.1f/10（低于要求的 7.0），请改写以提升质量。%s

**改写要求：**
- 深化核心观点，避免泛泛而谈
- 确保共识与分歧都有充分呈现
- 行动建议要具体、可执行
- 字数控制在 %d-%d 字
- 保持客观、专业的语调

**观点矩阵：**
共识：%s
分歧：%s
关键问题：%s
行动建议：%s

**原始报告（待改写）：**
%s

请直接输出改写后的报告正文，不要加标题或格式标记。`,
		score, issuesStr,
		g.cfg.MinWords, g.cfg.MaxWords,
		strings.Join(matrix.ConsensusPoints, "；"),
		strings.Join(matrix.DivergencePoints, "；"),
		strings.Join(matrix.KeyQuestions, "；"),
		strings.Join(matrix.ActionItems, "；"),
		originalSummary,
	)

	resp, err := g.llm.Complete(ctx, llm.Request{
		TaskType: llm.TaskReportGeneration,
		Messages: []llm.Message{
			{Role: "system", Content: g.reportSystemPrompt()},
			{Role: "user", Content: rewritePrompt},
		},
		MaxTokens:   2048,
		Temperature: 0.8,
		Module:      "report",
		ReferenceID: discussionID,
	})
	if err != nil {
		return "", 0, 0, err
	}
	return resp.Content, resp.TotalTokens, resp.CostUSD, nil
}

// ruleCheck applies fast rule-based quality checks.
func (g *Generator) ruleCheck(summary string) []string {
	var issues []string
	wordCount := countChineseWords(summary)

	if wordCount < g.cfg.MinWords {
		issues = append(issues, fmt.Sprintf("字数不足（%d < %d）", wordCount, g.cfg.MinWords))
	}
	if wordCount > g.cfg.MaxWords {
		issues = append(issues, fmt.Sprintf("字数超出（%d > %d）", wordCount, g.cfg.MaxWords))
	}
	if strings.Contains(summary, "作为AI") || strings.Contains(summary, "作为一个AI") {
		issues = append(issues, "包含AI身份暴露语句")
	}
	if len(summary) == 0 {
		issues = append(issues, "报告内容为空")
	}
	return issues
}

// scoreConnectionCandidates ranks agents for connection recommendation.
func (g *Generator) scoreConnectionCandidates(
	candidates []ConnectionCandidate,
	messages []DiscussionMessage,
) []RecommendedAgent {
	// Build insight counts from message analysis
	insightCounts := buildInsightCounts(messages)

	scored := make([]RecommendedAgent, 0, len(candidates))
	for _, c := range candidates {
		insight := insightCounts[c.AgentID]
		insightScore := normalizeInsight(insight)

		// 互补性：与 0.5 的差异越小越好（适中最优）
		compScore := 1.0 - 2.0*abs(c.Complementarity-0.5)
		if compScore < 0 {
			compScore = 0
		}

		breakdown := ScoreBreakdown{
			InsightScore:       insightScore,
			ComplementaryScore: compScore,
			CollabScore:        c.CollabSignal,
			ActivityScore:      c.ActivityScore,
		}

		// 加权综合分
		finalScore := insightScore*0.40 +
			compScore*0.30 +
			c.CollabSignal*0.20 +
			c.ActivityScore*0.10

		reasons := buildReasons(breakdown)

		scored = append(scored, RecommendedAgent{
			AgentID:        c.AgentID,
			AnonID:         c.AnonID,
			FinalScore:     finalScore,
			ScoreBreakdown: breakdown,
			Reasons:        reasons,
		})
	}

	// Sort descending by final score
	sortRecommendedAgents(scored)

	max := g.cfg.MaxRecommendations
	if max > len(scored) {
		max = len(scored)
	}
	return scored[:max]
}

// =============================================================================
// Prompt builders
// =============================================================================

func (g *Generator) buildOpinionExtractionPrompt(topic Topic, keyPoints []KeyPointEntry) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**讨论主题：** %s\n\n", topic.Title))
	sb.WriteString(fmt.Sprintf("**主题描述：** %s\n\n", topic.Description))
	sb.WriteString("**各角色各轮次核心观点：**\n")

	for round := 1; round <= 4; round++ {
		sb.WriteString(fmt.Sprintf("\n【第%d轮】\n", round))
		for _, kp := range keyPoints {
			if kp.RoundNumber == round {
				sb.WriteString(fmt.Sprintf("- %s（%s）：%s\n", kp.AnonID, roleLabel(kp.Role), kp.KeyPoint))
			}
		}
	}

	sb.WriteString(`
请基于上述观点提炼以下内容，以JSON格式返回：
{
  "consensus_points": ["共识观点1", "共识观点2", ...],
  "divergence_points": ["分歧点1", "分歧点2", ...],
  "key_questions": ["关键问题1", "关键问题2", ...],
  "action_items": ["行动建议1", "行动建议2", ...],
  "blind_spots": ["盲点/未考虑因素1", ...]
}

要求：
- 每类 2-4 条
- 用简洁的中文表达，每条不超过 50 字
- 基于观点提炼，不要捏造
`)
	return sb.String()
}

func (g *Generator) buildSummaryPrompt(topic Topic, matrix *OpinionMatrix) string {
	matrixJSON, _ := json.MarshalIndent(matrix, "", "  ")
	return fmt.Sprintf(`请基于以下讨论结果，撰写一份专业的讨论分析报告。

**主题：** %s
**背景：** %s

**讨论提炼结果：**
%s

**报告要求：**
1. 字数：%d-%d字
2. 结构：简短引言 → 核心共识 → 主要分歧与争议 → 关键疑问 → 行动建议 → 值得关注的盲点
3. 语调：客观专业，有深度，避免泛泛而谈
4. 价值导向：帮助读者从多维视角理解问题，获得实际启发
5. 禁止：不要捏造讨论中没有出现的观点

请直接输出报告正文。`,
		topic.Title,
		topic.Description,
		string(matrixJSON),
		800, 1200,
	)
}

func (g *Generator) reportSystemPrompt() string {
	return `你是一位专业的商业分析师和讨论记录员。你的任务是将多方讨论的核心观点提炼成高质量的分析报告。
报告应当客观呈现各方立场，不偏袒任何一方，同时提供有建设性的综合分析。
使用清晰的中文，段落结构分明，有洞见，对读者具有实际价值。`
}

// =============================================================================
// Parsing
// =============================================================================

func parseOpinionMatrix(content string) (*OpinionMatrix, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) >= 3 {
			content = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	var matrix OpinionMatrix
	if err := json.Unmarshal([]byte(content), &matrix); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return &matrix, nil
}

// =============================================================================
// Helpers
// =============================================================================

func countChineseWords(s string) int {
	// For Chinese text, count runes as "words"
	return utf8.RuneCountInString(s)
}

func roleLabel(role string) string {
	labels := map[string]string{
		"questioner":   "质疑者",
		"supporter":    "支持者",
		"supplementer": "补充者",
		"inquirer":     "提问者",
	}
	if l, ok := labels[role]; ok {
		return l
	}
	return role
}

func buildInsightCounts(messages []DiscussionMessage) map[string]int {
	counts := make(map[string]int)
	for _, msg := range messages {
		// Heuristic: longer key points and higher confidence = more insight
		if len(msg.KeyPoint) > 20 && msg.Confidence > 0.7 {
			counts[msg.AgentID]++
		}
	}
	return counts
}

func normalizeInsight(count int) float64 {
	if count <= 0 {
		return 0
	}
	// Cap at 5, normalize to 0-1
	if count > 5 {
		count = 5
	}
	return float64(count) / 5.0
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func buildReasons(b ScoreBreakdown) []string {
	var reasons []string
	if b.InsightScore > 0.6 {
		reasons = append(reasons, "提供了高价值洞见")
	}
	if b.ComplementaryScore > 0.6 {
		reasons = append(reasons, "背景互补，适合深度交流")
	}
	if b.CollabScore > 0.6 {
		reasons = append(reasons, "历史连接质量高")
	}
	if b.ActivityScore > 0.7 {
		reasons = append(reasons, "活跃度高，响应及时")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "综合评估推荐")
	}
	return reasons
}

func sortRecommendedAgents(agents []RecommendedAgent) {
	// Simple insertion sort (small N)
	for i := 1; i < len(agents); i++ {
		for j := i; j > 0 && agents[j].FinalScore > agents[j-1].FinalScore; j-- {
			agents[j], agents[j-1] = agents[j-1], agents[j]
		}
	}
}

// ---------------------------------------------------------------------------
// Repository interface – implemented by report/db
// ---------------------------------------------------------------------------

// Repository abstracts persistent storage for reports.
type Repository interface {
	// Save persists a new report and populates r.ID.
	Save(ctx context.Context, r *Report) error
	// FindByID retrieves a report by primary key.
	FindByID(ctx context.Context, id string) (*Report, error)
	// FindByDiscussionID retrieves the report for a discussion.
	FindByDiscussionID(ctx context.Context, discussionID string) (*Report, error)
	// UpdateUserRating stores a user's star rating and optional feedback.
	UpdateUserRating(ctx context.Context, id string, rating int, feedback string) error
}
