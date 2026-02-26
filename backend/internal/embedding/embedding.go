// Package embedding handles vector embedding generation and Qdrant read/write.
package embedding

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	qdrant "github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"

	"github.com/digital-twin-community/backend/internal/llm"
)

const (
	VectorDimension = 1024 // Voyage-3 dimension
)

// AgentPayload is stored alongside the vector in Qdrant.
type AgentPayload struct {
	AgentID        string   `json:"agent_id"`
	UserID         string   `json:"user_id"`
	AnonID         string   `json:"anon_id"`
	Industries     []string `json:"industries"`
	AgentType      string   `json:"agent_type"`
	QualityScore   float64  `json:"quality_score"`
	LastActiveUnix int64    `json:"last_active_unix"`
}

// Service manages vector operations.
type Service struct {
	qdrant           qdrant.PointsClient
	collections      qdrant.CollectionsClient
	llmGateway       *llm.Gateway
	agentsCollection string
	topicsCollection string
	logger           *zap.Logger
}

// NewService constructs an embedding Service.
// qdrantClient / collectionsClient may be nil; in that case operations are no-ops.
func NewService(
	qdrantClient qdrant.PointsClient,
	collectionsClient qdrant.CollectionsClient,
	gateway *llm.Gateway,
	agentsCollection string,
	topicsCollection string,
	logger *zap.Logger,
) *Service {
	return &Service{
		qdrant:           qdrantClient,
		collections:      collectionsClient,
		llmGateway:       gateway,
		agentsCollection: agentsCollection,
		topicsCollection: topicsCollection,
		logger:           logger,
	}
}

// EnsureCollections creates each named collection if it does not already exist.
// It is idempotent and safe to call on every startup.
func (s *Service) EnsureCollections(ctx context.Context, names []string) error {
	if s.collections == nil {
		return nil
	}

	resp, err := s.collections.List(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		return fmt.Errorf("list qdrant collections: %w", err)
	}

	existing := make(map[string]bool, len(resp.Collections))
	for _, c := range resp.Collections {
		existing[c.Name] = true
	}

	for _, name := range names {
		if existing[name] {
			s.logger.Info("qdrant collection already exists", zap.String("collection", name))
			continue
		}
		_, err = s.collections.Create(ctx, &qdrant.CreateCollection{
			CollectionName: name,
			VectorsConfig: &qdrant.VectorsConfig{
				Config: &qdrant.VectorsConfig_Params{
					Params: &qdrant.VectorParams{
						Size:     VectorDimension,
						Distance: qdrant.Distance_Cosine,
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("create qdrant collection %q: %w", name, err)
		}
		s.logger.Info("qdrant collection created", zap.String("collection", name))
	}
	return nil
}

// EmbedAndUpsertAgent generates an embedding for the agent's profile text
// and upserts the vector into Qdrant.
func (s *Service) EmbedAndUpsertAgent(ctx context.Context, agentID, text string, payload AgentPayload) (string, error) {
	vec, err := s.llmGateway.Embed(ctx, text, "embedding")
	if err != nil {
		return "", fmt.Errorf("embed agent text: %w", err)
	}
	if len(vec) == 0 {
		return "", fmt.Errorf("embedding returned empty vector")
	}

	pointID := uuid.NewString()

	if s.qdrant == nil {
		s.logger.Warn("qdrant client not configured; skipping agent upsert",
			zap.String("agent_id", agentID))
		return pointID, nil
	}

	_, err = s.qdrant.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.agentsCollection,
		Points: []*qdrant.PointStruct{
			{
				Id: &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: pointID}},
				Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{Data: vec},
				}},
				Payload: map[string]*qdrant.Value{
					"agent_id":         strVal(payload.AgentID),
					"user_id":          strVal(payload.UserID),
					"anon_id":          strVal(payload.AnonID),
					"agent_type":       strVal(payload.AgentType),
					"quality_score":    floatVal(payload.QualityScore),
					"last_active_unix": intVal(payload.LastActiveUnix),
					"industries":       strListVal(payload.Industries),
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("qdrant upsert agent: %w", err)
	}

	s.logger.Info("upserted agent vector",
		zap.String("agent_id", agentID),
		zap.String("point_id", pointID),
		zap.Int("dim", len(vec)),
	)
	return pointID, nil
}

// EmbedAndUpsertTopic generates an embedding for a topic and upserts it.
func (s *Service) EmbedAndUpsertTopic(ctx context.Context, topicID, text string) (string, error) {
	vec, err := s.llmGateway.Embed(ctx, text, "embedding")
	if err != nil {
		return "", fmt.Errorf("embed topic text: %w", err)
	}

	pointID := uuid.NewString()

	if s.qdrant == nil {
		s.logger.Warn("qdrant client not configured; skipping topic upsert",
			zap.String("topic_id", topicID))
		return pointID, nil
	}

	_, err = s.qdrant.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.topicsCollection,
		Points: []*qdrant.PointStruct{
			{
				Id: &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: pointID}},
				Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{Data: vec},
				}},
				Payload: map[string]*qdrant.Value{
					"topic_id": strVal(topicID),
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("qdrant upsert topic: %w", err)
	}

	s.logger.Info("upserted topic vector",
		zap.String("topic_id", topicID),
		zap.String("point_id", pointID),
	)
	return pointID, nil
}

// SearchSimilarAgents performs ANN search in Qdrant and returns candidate
// agent IDs sorted by score descending.
// Returns nil, nil when the qdrant client is not configured.
func (s *Service) SearchSimilarAgents(
	ctx context.Context,
	topicVec []float32,
	limit uint64,
	scoreThreshold float32,
	activeWithinDays int,
) ([]SearchResult, error) {
	if s.qdrant == nil {
		s.logger.Debug("qdrant client not configured; returning empty search results")
		return nil, nil
	}

	req := &qdrant.SearchPoints{
		CollectionName: s.agentsCollection,
		Vector:         topicVec,
		Limit:          limit,
		ScoreThreshold: &scoreThreshold,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true},
		},
	}

	// Optionally filter agents that have been active recently.
	if activeWithinDays > 0 {
		cutoff := float64(time.Now().AddDate(0, 0, -activeWithinDays).Unix())
		req.Filter = &qdrant.Filter{
			Must: []*qdrant.Condition{
				{
					ConditionOneOf: &qdrant.Condition_Field{
						Field: &qdrant.FieldCondition{
							Key:   "last_active_unix",
							Range: &qdrant.Range{Gte: &cutoff},
						},
					},
				},
			},
		}
	}

	resp, err := s.qdrant.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("qdrant search: %w", err)
	}

	results := make([]SearchResult, 0, len(resp.Result))
	for _, pt := range resp.Result {
		p := payloadFromMap(pt.Payload)
		results = append(results, SearchResult{
			AgentID: p.AgentID,
			AnonID:  p.AnonID,
			Score:   pt.Score,
			Payload: p,
		})
	}

	s.logger.Debug("qdrant search results",
		zap.Int("dim", len(topicVec)),
		zap.Uint64("limit", limit),
		zap.Int("hits", len(results)),
	)
	return results, nil
}

// SearchResult holds a single ANN search result.
type SearchResult struct {
	AgentID string
	AnonID  string
	Score   float32
	Payload AgentPayload
}

// =============================================================================
// Qdrant payload helpers
// =============================================================================

func strVal(s string) *qdrant.Value {
	return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: s}}
}

func floatVal(f float64) *qdrant.Value {
	return &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: f}}
}

func intVal(i int64) *qdrant.Value {
	return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: i}}
}

func strListVal(ss []string) *qdrant.Value {
	vals := make([]*qdrant.Value, len(ss))
	for i, s := range ss {
		vals[i] = strVal(s)
	}
	return &qdrant.Value{Kind: &qdrant.Value_ListValue{
		ListValue: &qdrant.ListValue{Values: vals},
	}}
}

func payloadFromMap(m map[string]*qdrant.Value) AgentPayload {
	var p AgentPayload
	if v, ok := m["agent_id"]; ok {
		p.AgentID = v.GetStringValue()
	}
	if v, ok := m["user_id"]; ok {
		p.UserID = v.GetStringValue()
	}
	if v, ok := m["anon_id"]; ok {
		p.AnonID = v.GetStringValue()
	}
	if v, ok := m["agent_type"]; ok {
		p.AgentType = v.GetStringValue()
	}
	if v, ok := m["quality_score"]; ok {
		p.QualityScore = v.GetDoubleValue()
	}
	if v, ok := m["last_active_unix"]; ok {
		p.LastActiveUnix = v.GetIntegerValue()
	}
	if v, ok := m["industries"]; ok {
		if lv := v.GetListValue(); lv != nil {
			for _, item := range lv.Values {
				p.Industries = append(p.Industries, item.GetStringValue())
			}
		}
	}
	return p
}
