// Package discussion implements the multi-round structured discussion engine
// for the 数字分身社区 (Digital Avatar Community) platform.
//
// State machine:
//
//	PENDING_MATCHING
//	  → ROUND_1_QUEUED → ROUND_1_RUNNING → ROUND_1_COMPLETED
//	  → ROUND_2_QUEUED → ROUND_2_RUNNING → ROUND_2_COMPLETED
//	  → ROUND_3_QUEUED → ROUND_3_RUNNING → ROUND_3_COMPLETED
//	  → ROUND_4_QUEUED → ROUND_4_RUNNING → ROUND_4_COMPLETED
//	  → REPORT_GENERATING → COMPLETED
//	  → DEGRADED  (fallback after 3 consecutive failures)
package discussion

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/digital-twin-community/backend/internal/llm"
)

// ---------------------------------------------------------------------------
// Domain constants
// ---------------------------------------------------------------------------

// DiscussionStatus enumerates all states in the discussion state machine.
type DiscussionStatus string

const (
	StatusPendingMatching DiscussionStatus = "PENDING_MATCHING"

	StatusRound1Queued    DiscussionStatus = "ROUND_1_QUEUED"
	StatusRound1Running   DiscussionStatus = "ROUND_1_RUNNING"
	StatusRound1Completed DiscussionStatus = "ROUND_1_COMPLETED"

	StatusRound2Queued    DiscussionStatus = "ROUND_2_QUEUED"
	StatusRound2Running   DiscussionStatus = "ROUND_2_RUNNING"
	StatusRound2Completed DiscussionStatus = "ROUND_2_COMPLETED"

	StatusRound3Queued    DiscussionStatus = "ROUND_3_QUEUED"
	StatusRound3Running   DiscussionStatus = "ROUND_3_RUNNING"
	StatusRound3Completed DiscussionStatus = "ROUND_3_COMPLETED"

	StatusRound4Queued    DiscussionStatus = "ROUND_4_QUEUED"
	StatusRound4Running   DiscussionStatus = "ROUND_4_RUNNING"
	StatusRound4Completed DiscussionStatus = "ROUND_4_COMPLETED"

	StatusReportGenerating DiscussionStatus = "REPORT_GENERATING"
	StatusCompleted        DiscussionStatus = "COMPLETED"
	StatusDegraded         DiscussionStatus = "DEGRADED"
)

// Role enumerates the four discussion roles.
type Role string

const (
	RoleQuestioner   Role = "questioner"   // 质疑者 – must find flaws, no vague statements
	RoleSupporter    Role = "supporter"    // 支持者 – must provide evidence
	RoleSupplementer Role = "supplementer" // 补充者 – expands perspectives
	RoleInquirer     Role = "inquirer"     // 提问者 – probes assumptions
)

// allRoles is the canonical iteration order for all four roles.
var allRoles = [4]Role{RoleQuestioner, RoleSupporter, RoleSupplementer, RoleInquirer}

// roundTemperature maps each round number (1-based) to its LLM temperature.
var roundTemperature = map[int]float32{
	1: 0.9,
	2: 0.8,
	3: 0.7,
	4: 0.5,
}

// similarityThreshold is the cosine similarity cut-off above which a rewrite
// instruction is injected.
const similarityThreshold = 0.85

// maxRounds is the total number of discussion rounds.
const maxRounds = 4

// maxErrorsBeforeDegraded is the threshold of consecutive failures that
// triggers the DEGRADED state.
const maxErrorsBeforeDegraded = 3

// ---------------------------------------------------------------------------
// Core domain structs
// ---------------------------------------------------------------------------

// Discussion is the top-level aggregate for a single multi-round discussion.
type Discussion struct {
	ID             string           `json:"id"`
	TopicID        string           `json:"topic_id"`
	TopicText      string           `json:"topic_text,omitempty"` // full topic text used in LLM prompts; set by the match worker
	Status         DiscussionStatus `json:"status"`
	CurrentRound   int              `json:"current_round"`
	ErrorCount     int              `json:"error_count,omitempty"`
	Participants   []Participant    `json:"participants"`
	Messages       []Message        `json:"messages"`
	IsDegraded     bool             `json:"is_degraded,omitempty"`
	DegradedReason string           `json:"degraded_reason,omitempty"`
}

// Message is a lightweight record stored inside Discussion.Messages.
type Message struct {
	RoundNum int    `json:"round_num"`
	AgentID  string `json:"agent_id"`
	Role     Role   `json:"role"`
	Content  string `json:"content"`
	KeyPoint string `json:"key_point"`
}

// Participant represents one digital avatar assigned to the discussion.
type Participant struct {
	AgentID       string             `json:"agent_id"`
	Role          Role               `json:"role"`
	AnonID        string             `json:"anon_id"`
	Industries    []string           `json:"industries"`
	Skills        []string           `json:"skills"`
	ThinkingStyle map[string]float64 `json:"thinking_style"`
	Background    string             `json:"background"`
}

// RoundMessage is the full output produced by one participant in one round.
type RoundMessage struct {
	RoundNum         int     `json:"round_num"` // set by the caller when saving; 1-based
	AgentID          string  `json:"agent_id"`
	Role             Role    `json:"role"`
	Content          string  `json:"content"`
	KeyPoint         string  `json:"key_point"`
	AddressedTo      Role    `json:"addressed_to"`
	Confidence       float64 `json:"confidence"`
	ModelUsed        string  `json:"model_used,omitempty"`
	PromptTokens     int     `json:"prompt_tokens,omitempty"`
	CompletionTokens int     `json:"completion_tokens,omitempty"`
	SimilarityToPrev float64 `json:"similarity_to_prev,omitempty"`
	WasRewritten     bool    `json:"was_rewritten,omitempty"`
}

// ---------------------------------------------------------------------------
// Engine
// ---------------------------------------------------------------------------

// Engine orchestrates the full discussion lifecycle.
type Engine struct {
	llm    *llm.Gateway
	logger *zap.Logger
}

// NewEngine constructs a ready-to-use Engine.
func NewEngine(gateway *llm.Gateway, logger *zap.Logger) *Engine {
	return &Engine{llm: gateway, logger: logger}
}

// RunDiscussion drives the entire state machine from PENDING_MATCHING through
// to COMPLETED (or DEGRADED).
func (e *Engine) RunDiscussion(ctx context.Context, discussion *Discussion, topic string) ([]*RoundMessage, error) {
	if discussion.Status != StatusPendingMatching {
		return nil, fmt.Errorf("discussion %s is not in PENDING_MATCHING state (current: %s)",
			discussion.ID, discussion.Status)
	}

	var allMessages []*RoundMessage

	for round := 1; round <= maxRounds; round++ {
		discussion.Status = queuedStatus(round)
		discussion.CurrentRound = round
		e.logger.Info("discussion round queued",
			zap.String("discussion_id", discussion.ID),
			zap.Int("round", round),
		)

		msgs, err := e.RunRound(ctx, discussion, round)
		if err != nil {
			discussion.IsDegraded = true
			discussion.DegradedReason = fmt.Sprintf("round %d failed: %v", round, err)
			discussion.Status = StatusDegraded
			e.logger.Error("discussion degraded",
				zap.String("discussion_id", discussion.ID),
				zap.Int("round", round),
				zap.Error(err),
			)
			return allMessages, fmt.Errorf("discussion degraded at round %d: %w", round, err)
		}

		allMessages = append(allMessages, msgs...)
		for _, m := range msgs {
			discussion.Messages = append(discussion.Messages, Message{
				RoundNum: round,
				AgentID:  m.AgentID,
				Role:     m.Role,
				Content:  m.Content,
				KeyPoint: m.KeyPoint,
			})
		}

		discussion.Status = completedStatus(round)
		e.logger.Info("discussion round completed",
			zap.String("discussion_id", discussion.ID),
			zap.Int("round", round),
			zap.Int("messages", len(msgs)),
		)
	}

	discussion.Status = StatusReportGenerating
	e.logger.Info("all rounds complete – entering report generation",
		zap.String("discussion_id", discussion.ID),
	)
	return allMessages, nil
}

// RunRound executes a single discussion round: fires parallel LLM calls for
// all four roles, checks for similarity, applies rewrites.
func (e *Engine) RunRound(ctx context.Context, discussion *Discussion, roundNum int) ([]*RoundMessage, error) {
	discussion.Status = runningStatus(roundNum)
	e.logger.Info("discussion round started",
		zap.String("discussion_id", discussion.ID),
		zap.Int("round", roundNum),
	)

	byRole := make(map[Role]*Participant, 4)
	for i := range discussion.Participants {
		p := &discussion.Participants[i]
		byRole[p.Role] = p
	}

	prevMessages := discussionMessagesToRoundMessages(discussion.Messages)
	topic := discussion.TopicText
	if topic == "" {
		topic = fmt.Sprintf("topic:%s", discussion.TopicID)
	}

	type result struct {
		role Role
		msg  *RoundMessage
		err  error
	}

	resultCh := make(chan result, 4)
	var wg sync.WaitGroup

	for _, role := range allRoles {
		participant, ok := byRole[role]
		if !ok {
			e.logger.Warn("no participant found for role – skipping",
				zap.String("discussion_id", discussion.ID),
				zap.String("role", string(role)),
			)
			continue
		}
		wg.Add(1)
		go func(p *Participant, r Role) {
			defer wg.Done()
			msg, err := e.runRoleCall(ctx, discussion, p, r, roundNum, topic, prevMessages)
			resultCh <- result{role: r, msg: msg, err: err}
		}(participant, role)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	roleOrder := map[Role]int{
		RoleQuestioner: 0, RoleSupporter: 1,
		RoleSupplementer: 2, RoleInquirer: 3,
	}
	var ordered [4]*RoundMessage
	var errs []error

	for res := range resultCh {
		if res.err != nil {
			errs = append(errs, fmt.Errorf("role %s: %w", res.role, res.err))
			discussion.ErrorCount++
			continue
		}
		idx := roleOrder[res.role]
		ordered[idx] = res.msg
	}

	if discussion.ErrorCount >= maxErrorsBeforeDegraded {
		return compactMessages(ordered),
			fmt.Errorf("error threshold reached (%d errors): %w",
				discussion.ErrorCount, errors.Join(errs...))
	}

	if len(errs) > 0 {
		e.logger.Warn("some roles failed in round – continuing with partial output",
			zap.String("discussion_id", discussion.ID),
			zap.Int("round", roundNum),
			zap.Errors("errors", errs),
		)
	}

	return compactMessages(ordered), nil
}

// runRoleCall executes the full pipeline for one role in one round.
func (e *Engine) runRoleCall(
	ctx context.Context,
	discussion *Discussion,
	participant *Participant,
	role Role,
	roundNum int,
	topic string,
	history []*RoundMessage,
) (*RoundMessage, error) {
	idempotencyKey := buildIdempotencyKey(discussion.ID, roundNum, role)
	temperature := roundTemperature[roundNum]

	messages := e.BuildRolePrompt(*participant, role, roundNum, topic, toRoundMessageSlice(history))

	prevSimilarity := e.checkSimilarity(ctx, "", history, role)
	if prevSimilarity > similarityThreshold {
		e.logger.Info("pre-call rewrite instruction injected",
			zap.String("discussion_id", discussion.ID),
			zap.String("role", string(role)),
			zap.Float64("similarity", prevSimilarity),
		)
		messages = appendRewriteInstruction(messages, prevSimilarity)
	}

	req := llm.Request{
		TaskType:       llm.TaskDiscussionRound,
		Messages:       messages,
		MaxTokens:      1024,
		Temperature:    temperature,
		ReferenceID:    discussion.ID,
		Module:         "discussion_engine",
		IdempotencyKey: idempotencyKey,
		UseCache:       true,
	}

	output, err := e.callWithRetry(ctx, req, discussion, roundNum, role)
	if err != nil {
		return nil, err
	}

	var postSimilarity float64
	var wasRewritten bool

	if roundNum > 1 {
		postSimilarity = e.checkSimilarity(ctx, output.Content, history, role)
		if postSimilarity > similarityThreshold {
			e.logger.Info("post-call similarity too high – requesting rewrite",
				zap.String("discussion_id", discussion.ID),
				zap.String("role", string(role)),
				zap.Float64("similarity", postSimilarity),
			)
			rewriteMessages := appendRewriteInstruction(messages, postSimilarity)
			rewriteReq := req
			rewriteReq.Messages = rewriteMessages
			rewriteReq.IdempotencyKey = idempotencyKey + ":rewrite"
			rewriteOutput, rewriteErr := e.callWithRetry(ctx, rewriteReq, discussion, roundNum, role)
			if rewriteErr == nil {
				output = rewriteOutput
				wasRewritten = true
			} else {
				e.logger.Warn("rewrite call failed – keeping original output",
					zap.String("discussion_id", discussion.ID),
					zap.String("role", string(role)),
					zap.Error(rewriteErr),
				)
			}
		}
	}

	addressedTo := resolveAddressedTo(output.AddressedTo, role)

	return &RoundMessage{
		AgentID:          participant.AgentID,
		Role:             role,
		Content:          output.Content,
		KeyPoint:         output.KeyPoint,
		AddressedTo:      addressedTo,
		Confidence:       output.Confidence,
		SimilarityToPrev: postSimilarity,
		WasRewritten:     wasRewritten,
	}, nil
}

// callWithRetry wraps the LLM gateway call with content-safety handling.
func (e *Engine) callWithRetry(
	ctx context.Context,
	req llm.Request,
	discussion *Discussion,
	roundNum int,
	role Role,
) (*llm.DiscussionOutput, error) {
	const maxContentSafetyRetries = 1

	for attempt := 0; attempt <= maxContentSafetyRetries; attempt++ {
		output, err := e.llm.CompleteDiscussionRound(ctx, req)
		if err == nil {
			return output, nil
		}

		if isContentSafetyError(err) {
			e.logger.Warn("content safety rejection – downgrading prompt",
				zap.String("discussion_id", discussion.ID),
				zap.Int("round", roundNum),
				zap.String("role", string(role)),
				zap.Int("attempt", attempt+1),
			)
			req.Messages = downgradeSafetyPrompt(req.Messages)
			req.IdempotencyKey = req.IdempotencyKey + ":safe"
			e.alertContentSafety(discussion.ID, roundNum, role)
			continue
		}

		return nil, fmt.Errorf("llm call failed (round %d, role %s): %w", roundNum, role, err)
	}

	return nil, fmt.Errorf("content safety rejection persisted after prompt downgrade (round %d, role %s)", roundNum, role)
}

func (e *Engine) alertContentSafety(discussionID string, roundNum int, role Role) {
	e.logger.Error("CONTENT_SAFETY_ALERT",
		zap.String("discussion_id", discussionID),
		zap.Int("round", roundNum),
		zap.String("role", string(role)),
		zap.Time("at", time.Now()),
	)
}

// ---------------------------------------------------------------------------
// Prompt Builder – 4-layer strategy
// ---------------------------------------------------------------------------

// BuildRolePrompt constructs the full 4-layer prompt for a participant.
func (e *Engine) BuildRolePrompt(
	participant Participant,
	role Role,
	roundNum int,
	topic string,
	history []RoundMessage,
) []llm.Message {
	return []llm.Message{
		{Role: "system", Content: buildLayer1SystemPrompt(role)},
		{Role: "user", Content: buildLayer2Context(participant, topic, history, roundNum)},
		{Role: "user", Content: buildLayer3Task(role, roundNum, history)},
		{Role: "user", Content: layer4FormatConstraint},
	}
}

const layer4FormatConstraint = `你必须只返回一个合法的 JSON 对象——不要 markdown 代码块，不要前言，不要多余文字。
JSON 必须严格符合以下格式：
{
  "content":      "<结构化回应，格式见下方要求>",
  "key_point":    "<一句话总结你的核心论点>",
  "addressed_to": "<exactly one of: questioner | supporter | supplementer | inquirer>",
  "confidence":   <float between 0.0 and 1.0>
}

content 字段必须严格按照以下结构化格式，用中文填写：

【立场】一句话表明你对议题的明确立场（支持/反对/有条件支持等）

【论据】
• 论据1：具体的数据、案例或逻辑推理
• 论据2：具体的数据、案例或逻辑推理
（2-3条，每条不超过80字）

【回应】（第2轮起必填，第1轮可省略）
针对"某人的某观点"：你的具体回应或反驳

【延伸问题】
一个值得进一步探讨的问题

注意：严格遵循上述模板，不要写成自然语言段落。`

// buildLayer1SystemPrompt returns the system instruction (Layer 1).
// All participants receive the same prompt — no fixed roles.
func buildLayer1SystemPrompt(role Role) string {
	_ = role // role is kept in signature for compatibility but not used in prompt
	return `你是数字分身社区平台上的一个数字分身（AI agent），正在参与一场多人结构化深度讨论。

核心要求：
• 全程用中文回复，采用结构化格式输出（见格式约束）。
• 基于你的行业背景和专业技能，给出有深度、有见地的观点。
• 每个论据必须具体、有数据或案例支撑，禁止空洞的套话。
• 不要重复别人说过的话，要贡献新的视角或更深入的分析。
• 如果不同意，直接说明理由；如果同意，在此基础上补充新论据。`
}

// buildLayer2Context assembles background tags + conversation history (Layer 2).
func buildLayer2Context(participant Participant, topic string, history []RoundMessage, roundNum int) string {
	var sb strings.Builder

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("讨论想法\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(topic)
	sb.WriteString("\n\n")

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("你的分身档案\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("匿名 ID        : %s\n", participant.AnonID))

	if len(participant.Industries) > 0 {
		sb.WriteString(fmt.Sprintf("行业领域       : %s\n", strings.Join(participant.Industries, ", ")))
	}
	if len(participant.Skills) > 0 {
		sb.WriteString(fmt.Sprintf("专业技能       : %s\n", strings.Join(participant.Skills, ", ")))
	}
	if len(participant.ThinkingStyle) > 0 {
		sb.WriteString("思维风格       :\n")
		for axis, score := range participant.ThinkingStyle {
			bar := buildScoreBar(score)
			sb.WriteString(fmt.Sprintf("  %-22s %s %.2f\n", axis, bar, score))
		}
	}
	if participant.Background != "" {
		sb.WriteString("\n背景简介:\n")
		sb.WriteString(participant.Background)
		sb.WriteString("\n")
	}

	const historyWindowRounds = 2
	if len(history) > 0 {
		sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("对话历史\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

		start := 0
		if roundNum > historyWindowRounds+1 {
			cutoff := len(history) - historyWindowRounds*4
			if cutoff > 0 {
				start = cutoff
			}
		}
		// Map agent IDs to numbered labels for anonymity
		agentNum := make(map[string]int)
		counter := 0
		for i := start; i < len(history); i++ {
			m := history[i]
			if _, ok := agentNum[m.AgentID]; !ok {
				counter++
				agentNum[m.AgentID] = counter
			}
			label := fmt.Sprintf("分身 %d", agentNum[m.AgentID])
			if m.AgentID == participant.AgentID {
				label = "你"
			}
			sb.WriteString(fmt.Sprintf("[%s]\n", label))
			sb.WriteString(m.Content)
			if m.KeyPoint != "" {
				sb.WriteString(fmt.Sprintf("\n  核心论点: %s", m.KeyPoint))
			}
			sb.WriteString("\n\n")
		}
	}
	return sb.String()
}

// buildLayer3Task constructs the per-round task instruction (Layer 3).
func buildLayer3Task(role Role, roundNum int, history []RoundMessage) string {
	_ = role // role kept for signature compatibility
	var sb strings.Builder

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("第 %d 轮任务\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n", roundNum))

	lastMsg := findLastMessage(history)

	switch roundNum {
	case 1:
		sb.WriteString("这是开场轮。基于你的专业背景，对这个想法给出你最核心的观点。\n")
		sb.WriteString("从第一句话开始就要直接、具体、有力。\n")
	case 2:
		sb.WriteString("这是展开轮。推进讨论——不要重复第一轮的内容。\n")
		if lastMsg != nil {
			sb.WriteString(fmt.Sprintf("请针对这个观点展开回应：「%s」\n", lastMsg.KeyPoint))
		}
	case 3:
		sb.WriteString("这是深化轮。挖掘讨论中浮现的核心分歧或关键问题。\n")
		if lastMsg != nil {
			sb.WriteString(fmt.Sprintf("请围绕这个未解决的问题深入分析：「%s」\n", lastMsg.KeyPoint))
		}
		sb.WriteString("不要重复之前的论点——探索边界情况、系统性影响或二阶效应。\n")
	case 4:
		sb.WriteString("这是总结轮。给出你最终的、最精炼的观点。\n")
		if lastMsg != nil {
			sb.WriteString(fmt.Sprintf("结合这个观点来总结你的立场：「%s」\n", lastMsg.KeyPoint))
		}
		sb.WriteString("承认讨论中最有力的反对意见，并解释为什么你的观点依然成立。\n")
	}

	return sb.String()
}

// findLastMessage returns the most recent message in history.
func findLastMessage(history []RoundMessage) *RoundMessage {
	if len(history) == 0 {
		return nil
	}
	return &history[len(history)-1]
}

func buildScoreBar(score float64) string {
	const total = 10
	filled := int(math.Round(score * total))
	if filled < 0 {
		filled = 0
	}
	if filled > total {
		filled = total
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", total-filled) + "]"
}


// ---------------------------------------------------------------------------
// Similarity Check
// ---------------------------------------------------------------------------

// CosineSimilarity computes cosine similarity between two float64 vectors.
func (e *Engine) CosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}

// checkSimilarity computes cosine similarity between current text and the most
// recent same-role message. When current is empty, compares the two most
// recent same-role messages (pre-check mode).
func (e *Engine) checkSimilarity(
	ctx context.Context,
	current string,
	previous []*RoundMessage,
	role Role,
) float64 {
	var lastMsg *RoundMessage
	for i := len(previous) - 1; i >= 0; i-- {
		if previous[i].Role == role {
			lastMsg = previous[i]
			break
		}
	}
	if lastMsg == nil {
		return 0
	}

	compareText := current
	if compareText == "" {
		var secondLast *RoundMessage
		for i := len(previous) - 1; i >= 0; i-- {
			if previous[i].Role == role && previous[i] != lastMsg {
				secondLast = previous[i]
				break
			}
		}
		if secondLast == nil {
			return 0
		}
		compareText = secondLast.Content
	}

	vecA, err := e.llm.Embed(ctx, compareText, "discussion_similarity")
	if err != nil {
		e.logger.Warn("embedding failed (compare text)",
			zap.String("role", string(role)), zap.Error(err))
		return 0
	}
	vecB, err := e.llm.Embed(ctx, lastMsg.Content, "discussion_similarity")
	if err != nil {
		e.logger.Warn("embedding failed (last message)",
			zap.String("role", string(role)), zap.Error(err))
		return 0
	}
	return e.CosineSimilarity(float32SliceToFloat64(vecA), float32SliceToFloat64(vecB))
}

// ---------------------------------------------------------------------------
// State machine transition helpers
// ---------------------------------------------------------------------------

func queuedStatus(round int) DiscussionStatus {
	switch round {
	case 1:
		return StatusRound1Queued
	case 2:
		return StatusRound2Queued
	case 3:
		return StatusRound3Queued
	case 4:
		return StatusRound4Queued
	}
	return StatusDegraded
}

func runningStatus(round int) DiscussionStatus {
	switch round {
	case 1:
		return StatusRound1Running
	case 2:
		return StatusRound2Running
	case 3:
		return StatusRound3Running
	case 4:
		return StatusRound4Running
	}
	return StatusDegraded
}

func completedStatus(round int) DiscussionStatus {
	switch round {
	case 1:
		return StatusRound1Completed
	case 2:
		return StatusRound2Completed
	case 3:
		return StatusRound3Completed
	case 4:
		return StatusRound4Completed
	}
	return StatusDegraded
}

// ---------------------------------------------------------------------------
// Prompt mutation helpers
// ---------------------------------------------------------------------------

func appendRewriteInstruction(messages []llm.Message, similarity float64) []llm.Message {
	instruction := fmt.Sprintf(
		`必须重写：你最近的输出与上一轮的回复语义相似度达 %.0f%%（阈值 %.0f%%）。
你必须大幅改变回复内容：
• 引入你之前未使用过的不同角度或论据。
• 完全更换你的开头句子。
• 如果之前在论证，现在试着解构其中一个前提假设；如果之前在质疑，提出一个具体的替代方案。
• 内容必须与你上一轮的回复有实质性的区别。`,
		similarity*100, similarityThreshold*100,
	)
	return append(messages, llm.Message{Role: "user", Content: instruction})
}

func downgradeSafetyPrompt(messages []llm.Message) []llm.Message {
	const safetyPreamble = "Note: Please ensure all responses are respectful, constructive, " +
		"and avoid inflammatory language. Focus on intellectual substance rather than personal criticism.\n\n"

	aggressive := []string{
		"MUST attack", "attack it directly",
		"zero in on", "most damaging",
		"Devil's Advocate", "FORBIDDEN",
	}

	moderated := make([]llm.Message, len(messages))
	copy(moderated, messages)

	for i, m := range moderated {
		content := m.Content
		if m.Role == "system" && i == 0 {
			content = safetyPreamble + content
		}
		for _, phrase := range aggressive {
			content = strings.ReplaceAll(content, phrase, "should carefully examine")
		}
		moderated[i].Content = content
	}
	return moderated
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

func buildIdempotencyKey(discussionID string, roundNum int, role Role) string {
	return fmt.Sprintf("%s:round%d:%s", discussionID, roundNum, string(role))
}

func resolveAddressedTo(raw string, speakerRole Role) Role {
	switch Role(strings.ToLower(strings.TrimSpace(raw))) {
	case RoleQuestioner:
		return RoleQuestioner
	case RoleSupporter:
		return RoleSupporter
	case RoleSupplementer:
		return RoleSupplementer
	case RoleInquirer:
		return RoleInquirer
	}
	switch speakerRole {
	case RoleQuestioner:
		return RoleSupporter
	case RoleSupporter:
		return RoleQuestioner
	case RoleSupplementer:
		return RoleInquirer
	case RoleInquirer:
		return RoleSupplementer
	}
	return RoleQuestioner
}

func isContentSafetyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, phrase := range []string{
		"content_policy", "content policy", "safety", "moderation",
		"harmful", "violates", "flagged", "filtered",
	} {
		if strings.Contains(msg, phrase) {
			return true
		}
	}
	return false
}

func compactMessages(ordered [4]*RoundMessage) []*RoundMessage {
	out := make([]*RoundMessage, 0, 4)
	for _, m := range ordered {
		if m != nil {
			out = append(out, m)
		}
	}
	return out
}

func toRoundMessageSlice(in []*RoundMessage) []RoundMessage {
	out := make([]RoundMessage, 0, len(in))
	for _, m := range in {
		if m != nil {
			out = append(out, *m)
		}
	}
	return out
}

func discussionMessagesToRoundMessages(msgs []Message) []*RoundMessage {
	out := make([]*RoundMessage, 0, len(msgs))
	for i := range msgs {
		m := &msgs[i]
		out = append(out, &RoundMessage{
			AgentID:  m.AgentID,
			Role:     m.Role,
			Content:  m.Content,
			KeyPoint: m.KeyPoint,
		})
	}
	return out
}

func float32SliceToFloat64(in []float32) []float64 {
	out := make([]float64, len(in))
	for i, v := range in {
		out[i] = float64(v)
	}
	return out
}

// ---------------------------------------------------------------------------
// Repository interface – implemented by discussion/db
// ---------------------------------------------------------------------------

// Repository abstracts persistent storage for discussions and their messages.
type Repository interface {
	// Create persists a new Discussion record.
	Create(ctx context.Context, d *Discussion) error
	// FindByID retrieves a Discussion by primary key (messages not populated).
	FindByID(ctx context.Context, id string) (*Discussion, error)
	// FindByTopicID retrieves the Discussion for a given topic.
	FindByTopicID(ctx context.Context, topicID string) (*Discussion, error)
	// UpdateStatus updates the discussion status.
	UpdateStatus(ctx context.Context, id string, status DiscussionStatus) error
	// SaveMessage persists one RoundMessage produced in the given round.
	SaveMessage(ctx context.Context, discussionID string, roundNum int, msg *RoundMessage) error
	// FindMessages returns all messages for a discussion in round/role order.
	FindMessages(ctx context.Context, discussionID string) ([]*RoundMessage, error)
	// SaveAnonMappings inserts anon_id → agent_id records for privacy audit.
	SaveAnonMappings(ctx context.Context, discussionID string, participants []Participant) error
}
