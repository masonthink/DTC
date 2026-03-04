# Test Case Designer - Agent Memory

## Project: Digital Twin Community (DTC)

### Architecture
- Backend: Go 1.22 + Echo v4, deployed in Docker on 45.32.57.146
- DB: PostgreSQL 16 (container: dtc-postgres, user: dtc, db: digital_twin_community)
- Redis 7 (container: dtc-redis, password-protected)
- Qdrant vector DB (container: dtc-qdrant, ports 6333/6334)
- Asynq for async tasks (queues: discussion:high, llm:standard, notification, report:generate)
- Frontend: Next.js 14 on Vercel at concors.cc

### API Routes (prefix: /api/v1)
- Auth: POST /auth/register, /auth/login, /auth/refresh (public)
- Users: GET /users/me, POST /users/fcm-token (protected)
- Agents: POST/GET /agents, GET/PUT /agents/:id (protected)
- Topics: POST/GET /topics, GET/DELETE /topics/:id (protected)
- Discussions: GET /discussions/:id, /discussions/:id/messages (protected)
- Reports: GET /reports/:id, POST /reports/:id/rating (protected)
- Connections: POST/GET /connections, POST /connections/:id/respond, GET /connections/:id/contacts

### Known Bugs (found 2026-03-02)
1. **httpError mapping gap**: `fmt.Errorf("agent not found")`, `"cannot cancel..."`, `"password must be..."` return 500 instead of 400/404 because they don't match sentinel errors or `isValidationError` patterns. See `/backend/internal/api/errors.go`.
2. **Voyage API 401**: Embedding service returns 401. All agents have NULL qdrant_point_id. Vector matching is non-functional; system falls back to seed agents.
3. **Qdrant healthcheck false-negative**: Container reports "unhealthy" because healthcheck uses `curl` which isn't installed in the Qdrant image. Functionally fine.

### Testing Patterns
- Server deployment path: /opt/digital-twin/
- .env file: /opt/digital-twin/.env.prod
- Redis password stored in .env.prod as REDIS_PASSWORD
- Production logger is zap.NewProduction() - Debug logs suppressed
- Scheduler repair function runs but produces no output if nothing to repair
- Seed agents use IDs: 00000000-0000-0000-0000-00000000000{1-4}
- DB column name: `round_number` (not `round_num` as in Go code)

### Data Consistency Notes
- Completed discussions should have 16 messages (4 rounds x 4 roles)
- Some completed discussions have fewer messages (degraded but still completed)
- `round_1_queued` stuck discussions: tasks archived due to duplicate key constraint on retries
