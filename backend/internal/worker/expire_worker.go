package worker

import (
	"context"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/digital-twin-community/backend/internal/connection"
)

// ExpireConnectionsWorker bulk-expires pending connections whose deadline has passed.
type ExpireConnectionsWorker struct {
	connRepo connection.Repository
	logger   *zap.Logger
}

// NewExpireConnectionsWorker constructs an ExpireConnectionsWorker.
func NewExpireConnectionsWorker(connRepo connection.Repository, logger *zap.Logger) *ExpireConnectionsWorker {
	return &ExpireConnectionsWorker{connRepo: connRepo, logger: logger}
}

// Handle processes the "connection:expire" task.
// The task carries no payload; it simply sweeps all overdue pending connections.
func (w *ExpireConnectionsWorker) Handle(ctx context.Context, task *asynq.Task) error {
	n, err := w.connRepo.ExpirePending(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		w.logger.Info("expire_worker: connections expired", zap.Int64("count", n))
	}
	return nil
}
