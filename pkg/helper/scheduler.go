package helper

import (
	"context"
	"log/slog"

	"github.com/robfig/cron/v3"
)

// Scheduler gerencia a execução periódica de health checks
type Scheduler struct {
	cron   *cron.Cron
	helper *Helper
	logger *slog.Logger
}

// SchedulerConfig configura o scheduler
type SchedulerConfig struct {
	// HealthCheckCron define o cron expression para health checks (default: "*/10 * * * *" = a cada 10 min)
	HealthCheckCron string
}

// DefaultSchedulerConfig retorna a configuração padrão
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		HealthCheckCron: "*/10 * * * *",
	}
}

func checkScheduledHealthCheck(h *Helper, logger *slog.Logger) {
	ctx := context.Background()
	results := h.CheckAll(ctx)
	for name, result := range results {
		logger.Info("scheduled health check",
			"service", name,
			"status", result.Status,
		)
	}
}

// NewScheduler cria e inicia o scheduler de health checks
func NewScheduler(h *Helper, cfg SchedulerConfig) *Scheduler {
	if cfg.HealthCheckCron == "" {
		cfg.HealthCheckCron = "*/10 * * * *"
	}

	logger := h.logger
	c := cron.New()

	s := &Scheduler{
		cron:   c,
		helper: h,
		logger: logger,
	}
	checkScheduledHealthCheck(h, logger) // Executa o health check imediatamente ao iniciar
	c.AddFunc(cfg.HealthCheckCron, func() {
		checkScheduledHealthCheck(h, logger)
	})

	c.Start()
	logger.Info("scheduler started", "health_check_cron", cfg.HealthCheckCron)

	return s
}

// Stop interrompe o scheduler graciosamente
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("scheduler stopped")
}
