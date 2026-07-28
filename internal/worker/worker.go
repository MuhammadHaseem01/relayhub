package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"relayhub/internal/dispatch"
	"relayhub/internal/queue"
)

const (
	blockDuration = 5 * time.Second

	reclaimInterval = 60 * time.Second

	reclaimMinIdle = 90 * time.Second

	reclaimBatch = 50
)

type Pool struct {
	workerCount int
	queue       *queue.Queue
	executor    *dispatch.Executor
	logger      *slog.Logger
}

type Params struct {
	WorkerCount int
	Queue       *queue.Queue
	Executor    *dispatch.Executor
	Logger      *slog.Logger
}

func New(p Params) *Pool {
	count := p.WorkerCount
	if count <= 0 {
		count = 5
	}
	return &Pool{
		workerCount: count,
		queue:       p.Queue,
		executor:    p.Executor,
		logger:      p.Logger,
	}
}

func (p *Pool) Run(ctx context.Context) {
	p.logger.Info("worker pool starting", "workers", p.workerCount)

	for i := range p.workerCount {
		name := fmt.Sprintf("worker-%d", i)
		go p.runWorker(ctx, name)
	}
	go p.runReclaimer(ctx)

	<-ctx.Done()
	p.logger.Info("worker pool stopped")
}

func (p *Pool) runWorker(ctx context.Context, name string) {
	log := p.logger.With("consumer", name)
	log.Info("worker started")

	for {
		select {
		case <-ctx.Done():
			log.Info("worker stopped")
			return
		default:
		}

		msgs, err := p.queue.Dequeue(ctx, name, 1, blockDuration)
		if err != nil {
			if ctx.Err() != nil {
				log.Info("worker stopped")
				return
			}
			log.Error("worker: dequeue error", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		for _, msg := range msgs {
			p.processOne(ctx, msg, log)
		}
	}
}

func (p *Pool) processOne(ctx context.Context, msg queue.Message, log *slog.Logger) {
	job := msg.Job
	log = log.With("request_id", job.RequestID, "channel", job.Channel)
	log.Info("worker: processing job")

	_ = p.executor.Run(ctx, dispatch.Job{
		RequestID:        job.RequestID,
		TenantID:         job.TenantID,
		Channel:          job.Channel,
		Recipient:        job.Recipient,
		Message:          job.Message,
		DiscordRecipient: job.DiscordRecipient,
		EmailRecipient:   job.EmailRecipient,
	})
	if err := p.queue.Ack(ctx, msg.ID); err != nil {
		log.Error("worker: failed to ACK message — will be reclaimed",
			"stream_id", msg.ID, "error", err)
	} else {
		log.Info("worker: job ACKed", "stream_id", msg.ID)
	}
}

func (p *Pool) runReclaimer(ctx context.Context) {
	ticker := time.NewTicker(reclaimInterval)
	defer ticker.Stop()

	log := p.logger.With("role", "reclaimer")
	log.Info("reclaimer started")

	for {
		select {
		case <-ctx.Done():
			log.Info("reclaimer stopped")
			return
		case <-ticker.C:
			p.reclaim(ctx, log)
		}
	}
}

func (p *Pool) reclaim(ctx context.Context, log *slog.Logger) {
	msgs, err := p.queue.Reclaim(ctx, "reclaimer", reclaimMinIdle, reclaimBatch)
	if err != nil {
		log.Error("reclaimer: XAUTOCLAIM error", "error", err)
		return
	}
	if len(msgs) == 0 {
		return
	}
	log.Info("reclaimer: re-processing orphaned jobs", "count", len(msgs))
	for _, msg := range msgs {
		p.processOne(ctx, msg, log)
	}
}
