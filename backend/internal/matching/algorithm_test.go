package matching

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newMatcher(t *testing.T) *Matcher {
	t.Helper()
	return NewMatcher(zaptest.NewLogger(t))
}

// randomVec returns a normalised random float32 vector of dimension d.
func randomVec(d int, seed int64) []float32 {
	rng := rand.New(rand.NewSource(seed))
	v := make([]float32, d)
	var norm float64
	for i := range v {
		x := rng.Float32()*2 - 1
		v[i] = x
		norm += float64(x * x)
	}
	norm = math.Sqrt(norm)
	for i := range v {
		v[i] = float32(float64(v[i]) / norm)
	}
	return v
}

// buildPool constructs n diverse Candidate entries with random embeddings.
func buildPool(n, dim int) []Candidate {
	candidates := make([]Candidate, n)
	industries := []string{"fintech", "healthcare", "edtech", "saas", "ai", "retail"}
	styles := []string{styleAnalytical, styleCreative, styleCritical, styleCollaborative}

	for i := 0; i < n; i++ {
		dom := styles[i%len(styles)]
		cand := Candidate{
			AgentID:         fmt.Sprintf("agent-%04d", i),
			AnonID:          fmt.Sprintf("AGT-%04X", i),
			Embedding:       randomVec(dim, int64(i+1)),
			Industries:      []string{industries[i%len(industries)]},
			Skills:          []string{"general"},
			ExperienceYears: (i % 20) + 1,
			QualityScore:    float64(i%5) / 5.0,
			LastActiveAt:    time.Now().Add(-time.Duration(i) * time.Hour * 24),
			ThinkingStyle: map[string]float64{
				dom:                  0.85,
				styles[(i+1)%4]:      0.4,
				styles[(i+2)%4]:      0.2,
				styles[(i+3)%4]:      0.1,
			},
			QuestionnaireMeta: map[string]interface{}{
				metaQuestioningAbility: float64((i % 10)) / 10.0,
			},
		}
		candidates[i] = cand
	}
	return candidates
}

// ─── Unit Tests ───────────────────────────────────────────────────────────────

// UnitTestCosineSimilarity verifies the cosine similarity implementation
// against known reference values.
func TestCosineSimilarity(t *testing.T) {
	m := newMatcher(t)

	tests := []struct {
		name string
		a, b []float32
		want float64
		tol  float64
	}{
		{
			name: "identical vectors → 1.0",
			a:    []float32{1, 0, 0},
			b:    []float32{1, 0, 0},
			want: 1.0,
			tol:  1e-9,
		},
		{
			name: "orthogonal vectors → 0.0",
			a:    []float32{1, 0},
			b:    []float32{0, 1},
			want: 0.0,
			tol:  1e-9,
		},
		{
			name: "opposite vectors → -1.0",
			a:    []float32{1, 0},
			b:    []float32{-1, 0},
			want: -1.0,
			tol:  1e-9,
		},
		{
			name: "general case",
			a:    []float32{3, 4},
			b:    []float32{4, 3},
			want: 0.96, // 3*4 + 4*3 = 24; |a|=|b|=5; 24/25=0.96
			tol:  1e-9,
		},
		{
			name: "empty vectors → 0.0",
			a:    []float32{},
			b:    []float32{},
			want: 0.0,
			tol:  1e-9,
		},
		{
			name: "length mismatch → 0.0",
			a:    []float32{1, 2},
			b:    []float32{1},
			want: 0.0,
			tol:  1e-9,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := m.cosineSimilarity(tc.a, tc.b)
			if math.Abs(got-tc.want) > tc.tol {
				t.Errorf("got %.6f, want %.6f (tol %.2e)", got, tc.want, tc.tol)
			}
		})
	}
}

// UnitTestMMRSelectIndustryCap verifies the hard constraint: at most 2 agents
// from the same industry.
func TestMMRSelectIndustryCap(t *testing.T) {
	m := newMatcher(t)
	const dim = 8
	topicVec := randomVec(dim, 42)

	// Create 6 candidates all from the same industry.
	same := make([]Candidate, 6)
	for i := range same {
		same[i] = Candidate{
			AgentID:       fmt.Sprintf("same-%d", i),
			Embedding:     randomVec(dim, int64(100+i)),
			Industries:    []string{"fintech"},
			ThinkingStyle: map[string]float64{styleAnalytical: 0.8, styleCreative: 0.3, styleCritical: 0.4, styleCollaborative: 0.1},
			LastActiveAt:  time.Now(),
			QualityScore:  0.8,
		}
	}
	// Add 2 agents from different industries so we can always fill 4 slots.
	other := []Candidate{
		{AgentID: "other-0", Embedding: randomVec(dim, 200), Industries: []string{"healthcare"}, ThinkingStyle: map[string]float64{styleCreative: 0.9}, LastActiveAt: time.Now()},
		{AgentID: "other-1", Embedding: randomVec(dim, 201), Industries: []string{"edtech"}, ThinkingStyle: map[string]float64{styleCritical: 0.9}, LastActiveAt: time.Now()},
	}
	all := append(same, other...)
	m.computeScores(all, topicVec)

	// Pool ≥ ColdStartPoolThreshold: use MMR path.
	// To avoid degraded path in a test we need 1000+ candidates.
	// So we call mmrSelect directly.
	selected := m.mmrSelect(all, topicVec, 4)

	finCount := 0
	for _, c := range selected {
		for _, ind := range c.Industries {
			if ind == "fintech" {
				finCount++
			}
		}
	}
	if finCount > MaxSameIndustry {
		t.Errorf("industry cap violated: %d > %d agents from fintech", finCount, MaxSameIndustry)
	}
}

// UnitTestMMRSelectDiversityConstraint verifies MinThinkingStyleTypes >= 3.
func TestMMRSelectDiversityConstraint(t *testing.T) {
	m := newMatcher(t)
	const dim = 8
	topicVec := randomVec(dim, 99)

	pool := buildPool(50, dim)
	m.computeScores(pool, topicVec)
	selected := m.mmrSelect(pool, topicVec, 4)

	covered := dominantStyles(selected)
	if len(covered) < MinThinkingStyleTypes {
		t.Errorf("thinking style variety constraint violated: only %d styles covered (need %d)",
			len(covered), MinThinkingStyleTypes)
	}
}

// UnitTestRoleAssignment verifies that 4 distinct roles are assigned to 4 agents
// and each agent receives exactly one role.
func TestRoleAssignment(t *testing.T) {
	m := newMatcher(t)
	pool := buildPool(4, 8)

	roles := m.assignRoles(pool)
	if len(roles) != 4 {
		t.Fatalf("expected 4 role assignments, got %d", len(roles))
	}

	seen := make(map[string]bool)
	seenAgents := make(map[string]bool)
	for _, ra := range roles {
		if seen[ra.Role] {
			t.Errorf("duplicate role assigned: %s", ra.Role)
		}
		seen[ra.Role] = true
		if seenAgents[ra.Candidate.AgentID] && len(pool) >= 4 {
			t.Errorf("agent %s assigned multiple roles with 4+ agents available", ra.Candidate.AgentID)
		}
		seenAgents[ra.Candidate.AgentID] = true
	}

	expected := []string{RoleQuestioner, RoleSupporter, RoleSupplementer, RoleInquirer}
	for _, role := range expected {
		if !seen[role] {
			t.Errorf("role %s was not assigned", role)
		}
	}
}

// UnitTestDegradedMatchFillsWithSeeds verifies that degradedMatch pads with
// generalist seed agents when the pool is too small.
func TestDegradedMatchFillsWithSeeds(t *testing.T) {
	m := newMatcher(t)
	const dim = 8
	topicVec := randomVec(dim, 77)

	// Only 2 real candidates above threshold.
	pool := []Candidate{
		{AgentID: "real-0", Embedding: topicVec /* sim=1.0 */, Industries: []string{"ai"}, ThinkingStyle: map[string]float64{styleAnalytical: 0.9}, LastActiveAt: time.Now()},
		{AgentID: "real-1", Embedding: topicVec /* sim=1.0 */, Industries: []string{"saas"}, ThinkingStyle: map[string]float64{styleCreative: 0.9}, LastActiveAt: time.Now()},
	}
	m.computeScores(pool, topicVec)

	selected := m.degradedMatch(pool, topicVec, 4)
	if len(selected) != 4 {
		t.Errorf("expected 4 selected agents, got %d", len(selected))
	}

	seedCount := 0
	for _, c := range selected {
		if len(c.Industries) == 1 && c.Industries[0] == generalistIndustry {
			seedCount++
		}
	}
	if seedCount != 2 {
		t.Errorf("expected 2 seed agents to fill gaps, got %d", seedCount)
	}
}

// UnitTestMatchReturnsDegradedWithSmallPool verifies that Match switches to the
// degraded algorithm when the pool is < ColdStartPoolThreshold.
func TestMatchReturnsDegradedWithSmallPool(t *testing.T) {
	m := newMatcher(t)
	pool := buildPool(50, 8) // < 1000 → degraded
	topicVec := randomVec(8, 1)

	result, err := m.Match(context.Background(), topicVec, pool, 4)
	if err != nil {
		t.Fatalf("Match returned error: %v", err)
	}
	if result.Algorithm != "degraded" {
		t.Errorf("expected algorithm=degraded, got %s", result.Algorithm)
	}
	if len(result.Participants) != 4 {
		t.Errorf("expected 4 participants, got %d", len(result.Participants))
	}
}

// UnitTestMatchMMRWithLargePool verifies that Match uses the MMR algorithm when
// the pool is >= ColdStartPoolThreshold.
func TestMatchMMRWithLargePool(t *testing.T) {
	m := newMatcher(t)
	pool := buildPool(ColdStartPoolThreshold+100, 16) // ≥ threshold → MMR
	topicVec := randomVec(16, 2)

	result, err := m.Match(context.Background(), topicVec, pool, 4)
	if err != nil {
		t.Fatalf("Match returned error: %v", err)
	}
	if result.Algorithm != "mmr" {
		t.Errorf("expected algorithm=mmr, got %s", result.Algorithm)
	}
}

// UnitTestMatchEmptyTopicVecError verifies that Match rejects an empty topic vector.
func TestMatchEmptyTopicVecError(t *testing.T) {
	m := newMatcher(t)
	pool := buildPool(10, 8)

	_, err := m.Match(context.Background(), nil, pool, 4)
	if err == nil {
		t.Error("expected error for empty topicEmbedding, got nil")
	}
}

// ─── Benchmarks ───────────────────────────────────────────────────────────────

func BenchmarkMMRSelect1000(b *testing.B) {
	m := &Matcher{logger: zap.NewNop()}
	pool := buildPool(1000, 1536)
	topicVec := randomVec(1536, 42)
	m.computeScores(pool, topicVec)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.mmrSelect(pool, topicVec, 4)
	}
}
