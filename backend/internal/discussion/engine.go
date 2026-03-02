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

const layer4FormatConstraint = `You MUST respond with ONLY a single valid JSON object — no markdown fences, no preamble, no trailing text.
The object must conform exactly to this schema:
{
  "content":      "<your full response, 100-300 words>",
  "key_point":    "<one sentence summarising your core argument>",
  "addressed_to": "<exactly one of: questioner | supporter | supplementer | inquirer>",
  "confidence":   <float between 0.0 and 1.0>
}`

// buildLayer1SystemPrompt returns the role-fixed instruction set (Layer 1).
func buildLayer1SystemPrompt(role Role) string {
	const base = `You are a digital avatar (数字分身) participating in a structured intellectual discussion on the 数字分身社区 platform.
You have been assigned a specific role that you MUST embody strictly throughout the conversation.
Do NOT produce vague or non-committal statements. Every claim must be precise and grounded.`

	switch role {
	case RoleQuestioner:
		return base + `

ROLE: 质疑者 (Questioner / Devil's Advocate)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
• Your PRIMARY mission is to find logical flaws, unsupported assumptions, and weak evidence in arguments.
• You MUST challenge every claim with a specific counter-example or falsifying condition.
• You are FORBIDDEN from agreeing wholesale with any prior statement without rigorous scrutiny.
• Identify the weakest link in the current argument chain and address it directly.
• Never use hedging language like "perhaps" — state your critique with precision.
• End every contribution with a pointed question that exposes a potential contradiction or gap.`

	case RoleSupporter:
		return base + `

ROLE: 支持者 (Supporter / Evidence Provider)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
• Your PRIMARY mission is to build the strongest possible case FOR the main thesis using concrete evidence.
• You MUST cite specific mechanisms, data patterns, or domain precedents — generic endorsements are prohibited.
• When the questioner raises a flaw, you MUST rebut it directly with counter-evidence rather than deflecting.
• Strengthen the weakest part of the thesis.
• Never use vague phrases like "studies show" without specifying what kind of evidence pattern applies.
• End every contribution with a question that invites deeper elaboration of supporting evidence.`

	case RoleSupplementer:
		return base + `

ROLE: 补充者 (Supplementer / Perspective Expander)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
• Your PRIMARY mission is to broaden the discussion by introducing adjacent dimensions not yet covered.
• You MUST add a new conceptual angle, cross-domain analogy, or systemic consideration in every response.
• Do NOT simply restate what the questioner or supporter said — add orthogonal value.
• Identify blind spots: what important factor is the discussion ignoring entirely?
• Vague "on the other hand" pivots are forbidden — your supplement must be specific and actionable.
• End every contribution with a question that draws attention to the newly introduced dimension.`

	case RoleInquirer:
		return base + `

ROLE: 提问者 (Inquirer / Assumption Prober)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
• Your PRIMARY mission is to surface and challenge the hidden assumptions underlying all positions.
• You MUST identify at least one unstated premise in the current exchange and probe it directly.
• Ask questions that reframe the problem rather than drill deeper into the existing framing.
• You are FORBIDDEN from stating opinions — your only tool is the Socratic question.
• Every question must be open-ended, non-leading, and designed to expose an assumption.
• End every contribution with your sharpest, most fundamental question of the round.`
	}

	return base
}

// buildLayer2Context assembles background tags + conversation history (Layer 2).
func buildLayer2Context(participant Participant, topic string, history []RoundMessage, roundNum int) string {
	var sb strings.Builder

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("DISCUSSION TOPIC\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(topic)
	sb.WriteString("\n\n")

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("YOUR AVATAR PROFILE\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("Anonymous ID   : %s\n", participant.AnonID))

	if len(participant.Industries) > 0 {
		sb.WriteString(fmt.Sprintf("Industries     : %s\n", strings.Join(participant.Industries, ", ")))
	}
	if len(participant.Skills) > 0 {
		sb.WriteString(fmt.Sprintf("Skills         : %s\n", strings.Join(participant.Skills, ", ")))
	}
	if len(participant.ThinkingStyle) > 0 {
		sb.WriteString("Thinking style :\n")
		for axis, score := range participant.ThinkingStyle {
			bar := buildScoreBar(score)
			sb.WriteString(fmt.Sprintf("  %-22s %s %.2f\n", axis, bar, score))
		}
	}
	if participant.Background != "" {
		sb.WriteString("\nBackground summary:\n")
		sb.WriteString(participant.Background)
		sb.WriteString("\n")
	}

	const historyWindowRounds = 2
	if len(history) > 0 {
		sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("CONVERSATION HISTORY\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

		start := 0
		if roundNum > historyWindowRounds+1 {
			cutoff := len(history) - historyWindowRounds*4
			if cutoff > 0 {
				start = cutoff
			}
		}
		for i := start; i < len(history); i++ {
			m := history[i]
			sb.WriteString(fmt.Sprintf("[%s]\n", strings.ToUpper(string(m.Role))))
			sb.WriteString(m.Content)
			if m.KeyPoint != "" {
				sb.WriteString(fmt.Sprintf("\n  KEY POINT: %s", m.KeyPoint))
			}
			sb.WriteString("\n\n")
		}
	}
	return sb.String()
}

// buildLayer3Task constructs the per-round task instruction (Layer 3).
func buildLayer3Task(role Role, roundNum int, history []RoundMessage) string {
	var sb strings.Builder

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("ROUND %d TASK\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n", roundNum))

	lastRelevant := findLastRelevantMessage(history, role)

	switch roundNum {
	case 1:
		sb.WriteString("This is the OPENING round. Introduce your position from your assigned role's perspective.\n")
		sb.WriteString("Be direct, specific, and concrete from the very first sentence.\n")
	case 2:
		sb.WriteString("This is the DEVELOPMENT round. Advance the argument — do NOT restate round 1.\n")
		if lastRelevant != nil {
			sb.WriteString(fmt.Sprintf("Respond SPECIFICALLY to this point: \"%s\"\n", lastRelevant.KeyPoint))
		}
	case 3:
		sb.WriteString("This is the DEEPENING round. Explore the core tension that has emerged.\n")
		if lastRelevant != nil {
			sb.WriteString(fmt.Sprintf("Address the unresolved tension around: \"%s\"\n", lastRelevant.KeyPoint))
		}
		sb.WriteString("Do not recycle earlier points — probe edge cases, systemic implications, or second-order effects.\n")
	case 4:
		sb.WriteString("This is the SYNTHESIS round. State your FINAL, most refined position.\n")
		if lastRelevant != nil {
			sb.WriteString(fmt.Sprintf("Synthesise your view in light of: \"%s\"\n", lastRelevant.KeyPoint))
		}
		sb.WriteString("Acknowledge the strongest opposing argument and explain why your view stands despite it.\n")
	}

	switch role {
	case RoleQuestioner:
		sb.WriteString("\nAs questioner: zero in on the single most damaging logical gap in the strongest argument so far.\n")
	case RoleSupporter:
		sb.WriteString("\nAs supporter: address the most forceful critique raised so far with the sharpest counter-evidence.\n")
	case RoleSupplementer:
		sb.WriteString("\nAs supplementer: name the ONE dimension most conspicuously absent from this discussion.\n")
	case RoleInquirer:
		sb.WriteString("\nAs inquirer: surface the deepest hidden assumption that, if false, would invalidate the dominant view.\n")
	}

	return sb.String()
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

func findLastRelevantMessage(history []RoundMessage, role Role) *RoundMessage {
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if m.AddressedTo == role || m.Role != role {
			return &history[i]
		}
	}
	return nil
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
		`REWRITE REQUIRED: Your most recent output is %.0f%% semantically similar to your previous round's output (threshold: %.0f%%).
You MUST substantially differentiate your response:
• Introduce a different angle or piece of evidence that you have NOT used before.
• Change your opening sentence entirely.
• If you were building a case, now deconstruct one of its assumptions; if questioning, propose a concrete alternative.
• The conceptual content must differ meaningfully from your previous response.`,
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
