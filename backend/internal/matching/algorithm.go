// Package matching implements the two-phase agent matching algorithm for 数字分身社区.
//
// Phase 1 (Recall) is performed externally via VectorStore (Qdrant ANN search).
// Phase 2 (MMR Reranking + Role Assignment) is performed here.
package matching

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	DefaultN               = 4
	ColdStartPoolThreshold = 1000
	ColdStartScoreThreshold = 0.50
	MaxSameIndustry        = 2
	MinThinkingStyleTypes  = 3

	mmrLambdaRelevance = 0.6
	mmrLambdaDiversity = 0.4

	weightRelevance = 0.50
	weightDiversity = 0.30
	weightActivity  = 0.10
	weightQuality   = 0.10

	activityHalfLifeDays = 15.0

	styleAnalytical    = "analytical"
	styleCreative      = "creative"
	styleCritical      = "critical"
	styleCollaborative = "collaborative"

	RoleQuestioner   = "questioner"
	RoleSupporter    = "supporter"
	RoleSupplementer = "supplementer"
	RoleInquirer     = "inquirer"

	metaQuestioningAbility = "questioning_ability"
	generalistIndustry     = "__generalist__"
)

// ---------------------------------------------------------------------------
// Public interfaces and types
// ---------------------------------------------------------------------------

// VectorStore abstracts Phase 1 Qdrant ANN recall.
type VectorStore interface {
	SearchAgents(
		ctx context.Context,
		topicVec []float32,
		limit int,
		scoreThreshold float32,
	) ([]Candidate, error)
}

// Candidate represents a single agent surfaced by Phase 1 recall.
type Candidate struct {
	AgentID           string
	AnonID            string
	Embedding         []float32
	Industries        []string
	Skills            []string
	ThinkingStyle     map[string]float64
	ExperienceYears   int
	QualityScore      float64
	LastActiveAt      time.Time
	QuestionnaireMeta map[string]interface{}

	// Computed during Phase 2.
	RelevanceScore float64
	DiversityScore float64
	ActivityScore  float64
	FinalScore     float64
}

// RoleAssignment pairs a matched Candidate with its assigned discussion role.
type RoleAssignment struct {
	Candidate Candidate
	Role      string
}

// MatchResult is the output of a successful Match call.
type MatchResult struct {
	Participants   []RoleAssignment
	Algorithm      string
	CandidateCount int
}

// ---------------------------------------------------------------------------
// Matcher
// ---------------------------------------------------------------------------

// Matcher orchestrates Phase 2 of the matching pipeline.
type Matcher struct {
	logger *zap.Logger
}

// NewMatcher constructs a Matcher.
func NewMatcher(logger *zap.Logger) *Matcher {
	return &Matcher{logger: logger}
}

// Match performs Phase 2 on pre-recalled candidates.
func (m *Matcher) Match(
	ctx context.Context,
	topicEmbedding []float32,
	candidates []Candidate,
	n int,
) (*MatchResult, error) {
	if len(topicEmbedding) == 0 {
		return nil, fmt.Errorf("matching: topicEmbedding must not be empty")
	}
	if n <= 0 {
		n = DefaultN
	}

	m.logger.Info("matching.Match started",
		zap.Int("candidates", len(candidates)),
		zap.Int("n", n),
	)

	m.computeScores(candidates, topicEmbedding)

	algo := "mmr"
	var selected []Candidate

	if len(candidates) < ColdStartPoolThreshold {
		m.logger.Warn("cold-start degradation triggered",
			zap.Int("poolSize", len(candidates)),
			zap.Int("threshold", ColdStartPoolThreshold),
		)
		algo = "degraded"
		selected = m.degradedMatch(candidates, topicEmbedding, n)
	} else {
		selected = m.mmrSelect(candidates, topicEmbedding, n)
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("matching: no agents could be selected from %d candidates", len(candidates))
	}

	roles := m.assignRoles(selected)

	m.logger.Info("matching.Match completed",
		zap.String("algorithm", algo),
		zap.Int("selected", len(selected)),
	)
	return &MatchResult{
		Participants:   roles,
		Algorithm:      algo,
		CandidateCount: len(candidates),
	}, nil
}

// ---------------------------------------------------------------------------
// computeScores
// ---------------------------------------------------------------------------

func (m *Matcher) computeScores(candidates []Candidate, topicVec []float32) {
	if len(candidates) == 0 {
		return
	}
	now := time.Now()

	maxQuality := 0.0
	for _, c := range candidates {
		if c.QualityScore > maxQuality {
			maxQuality = c.QualityScore
		}
	}
	if maxQuality == 0 {
		maxQuality = 1
	}

	for i := range candidates {
		c := &candidates[i]
		c.RelevanceScore = m.cosineSimilarity(c.Embedding, topicVec)
		daysSinceActive := now.Sub(c.LastActiveAt).Hours() / 24.0
		c.ActivityScore = math.Max(0, math.Min(1,
			math.Exp(-math.Log(2)*daysSinceActive/activityHalfLifeDays),
		))
		c.QualityScore = math.Max(0, math.Min(1, c.QualityScore/maxQuality))
	}
}

// ---------------------------------------------------------------------------
// mmrSelect
// ---------------------------------------------------------------------------

func (m *Matcher) mmrSelect(candidates []Candidate, topicVec []float32, n int) []Candidate {
	if len(candidates) == 0 {
		return nil
	}
	if n > len(candidates) {
		n = len(candidates)
	}

	selected := make([]Candidate, 0, n)
	remaining := make([]Candidate, len(candidates))
	copy(remaining, candidates)
	industryCount := make(map[string]int)

	for len(selected) < n && len(remaining) > 0 {
		bestIdx := -1
		bestMMR := math.Inf(-1)

		for i, cand := range remaining {
			if violatesIndustryCap(cand, industryCount) {
				continue
			}
			maxSim := 0.0
			for _, sel := range selected {
				sim := m.cosineSimilarity(cand.Embedding, sel.Embedding)
				if sim > maxSim {
					maxSim = sim
				}
			}
			mmrScore := mmrLambdaRelevance*cand.RelevanceScore - mmrLambdaDiversity*maxSim
			if mmrScore > bestMMR {
				bestMMR = mmrScore
				bestIdx = i
			}
		}

		if bestIdx == -1 {
			m.logger.Warn("mmrSelect: industry cap blocking all; relaxing",
				zap.Int("selectedSoFar", len(selected)))
			bestIdx = m.pickBestUnblocked(remaining, selected)
			if bestIdx == -1 {
				break
			}
		}

		chosen := remaining[bestIdx]
		chosen.DiversityScore = m.computeDiversityScore(chosen, selected)
		chosen.FinalScore = weightRelevance*chosen.RelevanceScore +
			weightDiversity*chosen.DiversityScore +
			weightActivity*chosen.ActivityScore +
			weightQuality*chosen.QualityScore

		selected = append(selected, chosen)
		for _, ind := range chosen.Industries {
			industryCount[ind]++
		}
		remaining[bestIdx] = remaining[len(remaining)-1]
		remaining = remaining[:len(remaining)-1]
	}

	return m.enforceThinkingStyleVariety(selected, remaining, industryCount, n)
}

func (m *Matcher) enforceThinkingStyleVariety(
	selected []Candidate,
	remaining []Candidate,
	industryCount map[string]int,
	n int,
) []Candidate {
	stylesCovered := dominantStyles(selected)
	if len(stylesCovered) >= MinThinkingStyleTypes {
		return selected
	}

	needed := missingStyles(stylesCovered)
	for _, style := range needed {
		if len(stylesCovered) >= MinThinkingStyleTypes {
			break
		}
		bestIdx := -1
		bestScore := math.Inf(-1)
		for i, cand := range remaining {
			if dominantStyle(cand) != style {
				continue
			}
			if cand.FinalScore > bestScore {
				bestScore = cand.FinalScore
				bestIdx = i
			}
		}
		if bestIdx == -1 {
			continue
		}
		swapIdx := m.swapTarget(selected, stylesCovered)
		if swapIdx == -1 {
			continue
		}
		evicted := selected[swapIdx]
		newcomer := remaining[bestIdx]
		for _, ind := range evicted.Industries {
			industryCount[ind]--
		}
		for _, ind := range newcomer.Industries {
			industryCount[ind]++
		}
		selected[swapIdx] = newcomer
		remaining[bestIdx] = remaining[len(remaining)-1]
		remaining = remaining[:len(remaining)-1]
		remaining = append(remaining, evicted)
		stylesCovered = dominantStyles(selected)
	}
	return selected
}

// ---------------------------------------------------------------------------
// degradedMatch
// ---------------------------------------------------------------------------

func (m *Matcher) degradedMatch(candidates []Candidate, topicVec []float32, n int) []Candidate {
	filtered := make([]Candidate, 0, len(candidates))
	for _, c := range candidates {
		if c.RelevanceScore >= ColdStartScoreThreshold {
			filtered = append(filtered, c)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].RelevanceScore > filtered[j].RelevanceScore
	})
	if len(filtered) > n {
		filtered = filtered[:n]
	}
	for i := range filtered {
		filtered[i].DiversityScore = m.computeDiversityScore(filtered[i], filtered[:i])
		filtered[i].FinalScore = weightRelevance*filtered[i].RelevanceScore +
			weightDiversity*filtered[i].DiversityScore +
			weightActivity*filtered[i].ActivityScore +
			weightQuality*filtered[i].QualityScore
	}
	if len(filtered) < n {
		needed := n - len(filtered)
		m.logger.Warn("degradedMatch: padding with generalist seed agents",
			zap.Int("have", len(filtered)), zap.Int("need", needed))
		filtered = append(filtered, m.generalistSeeds(needed)...)
	}
	return filtered
}

// seedAgentIDs are fixed valid UUIDs for the cold-start generalist seed agents.
var seedAgentIDs = [4]string{
	"00000000-0000-0000-0000-000000000001",
	"00000000-0000-0000-0000-000000000002",
	"00000000-0000-0000-0000-000000000003",
	"00000000-0000-0000-0000-000000000004",
}

// seedAnonNames are human-readable anonymous names for seed agents.
var seedAnonNames = [4]string{"Alex", "Jordan", "Morgan", "Riley"}

func (m *Matcher) generalistSeeds(count int) []Candidate {
	seeds := make([]Candidate, count)
	for i := range seeds {
		seeds[i] = Candidate{
			AgentID:    seedAgentIDs[i%4],
			AnonID:     seedAnonNames[i%4],
			Industries: []string{generalistIndustry},
			Skills:     []string{"general"},
			ThinkingStyle: map[string]float64{
				styleAnalytical: 0.25, styleCreative: 0.25,
				styleCritical: 0.25, styleCollaborative: 0.25,
			},
			ExperienceYears: 5,
			QualityScore:    0.5,
			LastActiveAt:    time.Now(),
			QuestionnaireMeta: map[string]interface{}{metaQuestioningAbility: 0.5},
			RelevanceScore: 0.50, DiversityScore: 1.0, ActivityScore: 1.0,
			FinalScore: weightRelevance*0.50 + weightDiversity*1.0 +
				weightActivity*1.0 + weightQuality*0.5,
		}
	}
	return seeds
}

// ---------------------------------------------------------------------------
// assignRoles
// ---------------------------------------------------------------------------

func (m *Matcher) assignRoles(selected []Candidate) []RoleAssignment {
	if len(selected) == 0 {
		return nil
	}
	roles := []string{RoleQuestioner, RoleSupporter, RoleSupplementer, RoleInquirer}
	if len(selected) >= 4 {
		return m.assignDistinct(selected, roles)
	}
	m.logger.Warn("assignRoles: fewer than 4 agents; roles may be shared",
		zap.Int("agentCount", len(selected)))
	picks := []Candidate{
		m.pickQuestioner(selected),
		m.pickSupporter(selected),
		m.pickSupplementer(selected),
		m.pickInquirer(selected),
	}
	assigned := make([]RoleAssignment, len(roles))
	for i, role := range roles {
		assigned[i] = RoleAssignment{Candidate: picks[i], Role: role}
	}
	return assigned
}

func (m *Matcher) assignDistinct(selected []Candidate, roles []string) []RoleAssignment {
	type entry struct{ roleIdx, candIdx int; score float64 }
	matrix := m.buildRoleScoreMatrix(selected)
	entries := make([]entry, 0, len(roles)*len(selected))
	for ri, scores := range matrix {
		for ci, s := range scores {
			entries = append(entries, entry{ri, ci, s})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].score > entries[j].score })

	usedRole := make([]bool, len(roles))
	usedCand := make([]bool, len(selected))
	result := make([]RoleAssignment, 0, len(roles))

	for _, e := range entries {
		if usedRole[e.roleIdx] || usedCand[e.candIdx] {
			continue
		}
		result = append(result, RoleAssignment{Candidate: selected[e.candIdx], Role: roles[e.roleIdx]})
		usedRole[e.roleIdx] = true
		usedCand[e.candIdx] = true
		if len(result) == len(roles) {
			break
		}
	}
	// Fill any unassigned (safety net)
	for ri, filled := range usedRole {
		if filled {
			continue
		}
		for ci, cand := range selected {
			if !usedCand[ci] {
				result = append(result, RoleAssignment{Candidate: cand, Role: roles[ri]})
				usedCand[ci] = true
				break
			}
		}
	}
	return result
}

func (m *Matcher) buildRoleScoreMatrix(selected []Candidate) [][]float64 {
	matrix := make([][]float64, 4)
	for i := range matrix {
		matrix[i] = make([]float64, len(selected))
	}
	for ci, c := range selected {
		matrix[0][ci] = questionerScore(c)
		matrix[1][ci] = supporterScore(c)
		matrix[2][ci] = supplementerScore(c, selected)
		matrix[3][ci] = inquirerScore(c)
	}
	return matrix
}

// ---------------------------------------------------------------------------
// Role scoring functions
// ---------------------------------------------------------------------------

func questionerScore(c Candidate) float64 {
	expNorm := math.Min(1.0, float64(c.ExperienceYears)/30.0)
	return 0.6*expNorm + 0.4*c.ThinkingStyle[styleCritical]
}

func supporterScore(c Candidate) float64 { return c.RelevanceScore }

func supplementerScore(c Candidate, all []Candidate) float64 {
	if len(all) <= 1 {
		return c.DiversityScore
	}
	totalSim, count := 0.0, 0
	for _, other := range all {
		if other.AgentID == c.AgentID {
			continue
		}
		totalSim += cosineSim32(c.Embedding, other.Embedding)
		count++
	}
	if count == 0 {
		return 0
	}
	return 1.0 - totalSim/float64(count)
}

func inquirerScore(c Candidate) float64 {
	if c.QuestionnaireMeta == nil {
		return 0
	}
	v, ok := c.QuestionnaireMeta[metaQuestioningAbility]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	default:
		return 0
	}
}

func (m *Matcher) pickQuestioner(cs []Candidate) Candidate {
	best := cs[0]
	for _, c := range cs[1:] {
		if questionerScore(c) > questionerScore(best) {
			best = c
		}
	}
	return best
}

func (m *Matcher) pickSupporter(cs []Candidate) Candidate {
	best := cs[0]
	for _, c := range cs[1:] {
		if supporterScore(c) > supporterScore(best) {
			best = c
		}
	}
	return best
}

func (m *Matcher) pickSupplementer(cs []Candidate) Candidate {
	best := cs[0]
	for _, c := range cs[1:] {
		if supplementerScore(c, cs) > supplementerScore(best, cs) {
			best = c
		}
	}
	return best
}

func (m *Matcher) pickInquirer(cs []Candidate) Candidate {
	best := cs[0]
	for _, c := range cs[1:] {
		if inquirerScore(c) > inquirerScore(best) {
			best = c
		}
	}
	return best
}

// ---------------------------------------------------------------------------
// Cosine similarity
// ---------------------------------------------------------------------------

func (m *Matcher) cosineSimilarity(a, b []float32) float64 { return cosineSim32(a, b) }

func cosineSim32(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		ai, bi := float64(a[i]), float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (m *Matcher) computeDiversityScore(cand Candidate, selected []Candidate) float64 {
	if len(selected) == 0 {
		return 1.0
	}
	maxSim := 0.0
	for _, sel := range selected {
		if sim := m.cosineSimilarity(cand.Embedding, sel.Embedding); sim > maxSim {
			maxSim = sim
		}
	}
	return 1.0 - maxSim
}

// ---------------------------------------------------------------------------
// Hard-constraint and style helpers
// ---------------------------------------------------------------------------

func violatesIndustryCap(cand Candidate, industryCount map[string]int) bool {
	for _, ind := range cand.Industries {
		if ind == generalistIndustry {
			continue
		}
		if industryCount[ind] >= MaxSameIndustry {
			return true
		}
	}
	return false
}

func dominantStyle(c Candidate) string {
	best, bestVal := "", math.Inf(-1)
	for k, v := range c.ThinkingStyle {
		if v > bestVal {
			bestVal = v
			best = k
		}
	}
	return best
}

func dominantStyles(cs []Candidate) map[string]bool {
	out := make(map[string]bool, len(cs))
	for _, c := range cs {
		if ds := dominantStyle(c); ds != "" {
			out[ds] = true
		}
	}
	return out
}

var allStyles = []string{styleAnalytical, styleCreative, styleCritical, styleCollaborative}

func missingStyles(covered map[string]bool) []string {
	var missing []string
	for _, s := range allStyles {
		if !covered[s] {
			missing = append(missing, s)
		}
	}
	return missing
}

func (m *Matcher) swapTarget(selected []Candidate, covered map[string]bool) int {
	styleCount := make(map[string]int)
	for _, c := range selected {
		styleCount[dominantStyle(c)]++
	}
	worstIdx, worstScore := -1, math.Inf(1)
	for i, c := range selected {
		if styleCount[dominantStyle(c)] <= 1 {
			continue
		}
		if c.FinalScore < worstScore {
			worstScore = c.FinalScore
			worstIdx = i
		}
	}
	return worstIdx
}

func (m *Matcher) pickBestUnblocked(remaining []Candidate, selected []Candidate) int {
	bestIdx, bestMMR := -1, math.Inf(-1)
	for i, cand := range remaining {
		maxSim := 0.0
		for _, sel := range selected {
			if sim := m.cosineSimilarity(cand.Embedding, sel.Embedding); sim > maxSim {
				maxSim = sim
			}
		}
		mmrScore := mmrLambdaRelevance*cand.RelevanceScore - mmrLambdaDiversity*maxSim
		if mmrScore > bestMMR {
			bestMMR = mmrScore
			bestIdx = i
		}
	}
	return bestIdx
}
