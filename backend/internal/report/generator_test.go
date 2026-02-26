package report

import (
	"testing"
)

// UnitTestRuleCheck verifies the fast rule-based quality checker.
func TestRuleCheck(t *testing.T) {
	g := &Generator{cfg: DefaultConfig()}

	tests := []struct {
		name     string
		summary  string
		wantIssues []string
	}{
		{
			name:    "empty summary",
			summary: "",
			wantIssues: []string{"报告内容为空"},
		},
		{
			name:    "AI identity exposure",
			summary: "作为AI，我认为这个方案是合理的，但需要更多数据支持。",
			wantIssues: []string{"包含AI身份暴露语句"},
		},
		{
			name:    "valid summary",
			summary: generateValidSummary(900),
			wantIssues: nil,
		},
		{
			name:    "too short",
			summary: "这个报告太短了。",
			wantIssues: []string{"字数不足"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issues := g.ruleCheck(tc.summary)
			if tc.wantIssues == nil && len(issues) != 0 {
				t.Errorf("expected no issues, got: %v", issues)
				return
			}
			for _, want := range tc.wantIssues {
				found := false
				for _, got := range issues {
					if len(got) >= len(want) && got[:len(want)] == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected issue containing %q, got %v", want, issues)
				}
			}
		})
	}
}

// UnitTestScoreConnectionCandidates verifies that recommendations are sorted
// by final_score descending and capped at MaxRecommendations.
func TestScoreConnectionCandidates(t *testing.T) {
	g := &Generator{cfg: DefaultConfig()}

	candidates := []ConnectionCandidate{
		{AgentID: "a1", AnonID: "AGT-001", InsightCount: 5, Complementarity: 0.5, CollabSignal: 0.9, ActivityScore: 0.8},
		{AgentID: "a2", AnonID: "AGT-002", InsightCount: 1, Complementarity: 0.2, CollabSignal: 0.3, ActivityScore: 0.4},
		{AgentID: "a3", AnonID: "AGT-003", InsightCount: 3, Complementarity: 0.5, CollabSignal: 0.7, ActivityScore: 0.6},
		{AgentID: "a4", AnonID: "AGT-004", InsightCount: 4, Complementarity: 0.6, CollabSignal: 0.8, ActivityScore: 0.9},
	}

	messages := []DiscussionMessage{
		{AgentID: "a1", KeyPoint: "非常有洞见的观点，值得深入探讨", Confidence: 0.9},
		{AgentID: "a3", KeyPoint: "从跨领域视角提供了补充", Confidence: 0.8},
	}

	recs := g.scoreConnectionCandidates(candidates, messages)

	if len(recs) > DefaultConfig().MaxRecommendations {
		t.Errorf("expected at most %d recommendations, got %d",
			DefaultConfig().MaxRecommendations, len(recs))
	}

	// Verify sorted descending
	for i := 1; i < len(recs); i++ {
		if recs[i].FinalScore > recs[i-1].FinalScore {
			t.Errorf("recommendations not sorted: recs[%d].score=%.3f > recs[%d].score=%.3f",
				i, recs[i].FinalScore, i-1, recs[i-1].FinalScore)
		}
	}
}

// UnitTestAggregateKeyPoints verifies that key points are extracted correctly
// from discussion messages.
func TestAggregateKeyPoints(t *testing.T) {
	g := &Generator{cfg: DefaultConfig()}

	messages := []DiscussionMessage{
		{AgentID: "a1", AnonID: "AGT-001", Role: "questioner", RoundNumber: 1, KeyPoint: "质疑观点1", Confidence: 0.8},
		{AgentID: "a2", AnonID: "AGT-002", Role: "supporter", RoundNumber: 1, KeyPoint: "支持观点1", Confidence: 0.9},
		{AgentID: "a3", AnonID: "AGT-003", Role: "supplementer", RoundNumber: 2, KeyPoint: "补充观点1", Confidence: 0.7},
		{AgentID: "a4", AnonID: "AGT-004", Role: "inquirer", RoundNumber: 2, KeyPoint: "提问观点1", Confidence: 0.6},
	}

	entries := g.aggregateKeyPoints(messages)
	if len(entries) != len(messages) {
		t.Errorf("expected %d key points, got %d", len(messages), len(entries))
	}
	for i, entry := range entries {
		if entry.KeyPoint != messages[i].KeyPoint {
			t.Errorf("entry[%d].KeyPoint mismatch: got %q, want %q", i, entry.KeyPoint, messages[i].KeyPoint)
		}
	}
}

// UnitTestParseOpinionMatrix verifies JSON parsing of the opinion matrix.
func TestParseOpinionMatrix(t *testing.T) {
	valid := `{
		"consensus_points": ["共识1", "共识2"],
		"divergence_points": ["分歧1"],
		"key_questions": ["问题1"],
		"action_items": ["行动1"],
		"blind_spots": ["盲点1"]
	}`

	matrix, err := parseOpinionMatrix(valid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matrix.ConsensusPoints) != 2 {
		t.Errorf("expected 2 consensus_points, got %d", len(matrix.ConsensusPoints))
	}
	if len(matrix.DivergencePoints) != 1 {
		t.Errorf("expected 1 divergence_point, got %d", len(matrix.DivergencePoints))
	}

	// Markdown code block wrapping
	wrapped := "```json\n" + valid + "\n```"
	matrix2, err := parseOpinionMatrix(wrapped)
	if err != nil {
		t.Fatalf("markdown-wrapped JSON should parse: %v", err)
	}
	if len(matrix2.ConsensusPoints) != 2 {
		t.Error("wrapped JSON: wrong consensus count")
	}

	// Invalid JSON
	_, err = parseOpinionMatrix("not json")
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// UnitTestSortRecommendedAgents verifies the insertion-sort order.
func TestSortRecommendedAgents(t *testing.T) {
	agents := []RecommendedAgent{
		{AgentID: "a", FinalScore: 0.3},
		{AgentID: "b", FinalScore: 0.9},
		{AgentID: "c", FinalScore: 0.6},
	}
	sortRecommendedAgents(agents)
	if agents[0].FinalScore != 0.9 {
		t.Errorf("expected first agent to have score 0.9, got %.1f", agents[0].FinalScore)
	}
	if agents[2].FinalScore != 0.3 {
		t.Errorf("expected last agent to have score 0.3, got %.1f", agents[2].FinalScore)
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// generateValidSummary returns a Chinese string of approximately n runes.
func generateValidSummary(n int) string {
	char := "这是一个有价值的分析观点，涵盖了多个维度的思考与讨论。"
	result := ""
	for len([]rune(result)) < n {
		result += char
	}
	return string([]rune(result)[:n])
}
