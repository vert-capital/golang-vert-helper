package helper

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/caiofariavert/golang_vert_helper/internal/adapters"
)

// RegisterRoutes registra todas as rotas do Vert Helper no router Gin do cliente.
//
// Parâmetros:
//   - router: o *gin.Engine já configurado pelo cliente
//   - db: conexão GORM para uso dos repositories
//   - middleware: middleware opcional (ex: autenticação). Passa nil para sem middleware.
//
// Rotas registradas em /api/helper/v1/:
//
//	POST /auth/                      	  → autentica e retorna token JWT (Bearer)
//	GET  /healthcare/                     → status geral de todos os serviços
//	GET  /healthcare/:name                → status de um serviço específico (query opcional: force_refresh=true)
//	GET  /actions/                        → lista actions (query: service_id)
//	GET  /actions/:slug                   → detalhe de uma action
//	POST /actions/:slug/execute           → executa uma action
//	GET  /workers/                        → lista workers do WorkerPool
//	GET  /workers/:id                     → detalhe de um worker
func (h *Helper) RegisterRoutes(router *gin.Engine, db *gorm.DB, middleware *gin.HandlerFunc) {
	handlers := adapters.NewHandlers(h.healthService, h.actionService, h.authService, h.workerPool)

	group := router.Group("/api/helper/v1")
	group.POST("/auth/", handlers.AuthLogin)

	if err := h.authService.ProvisionDefaultUserFromEnv(context.Background()); err != nil {
		h.logger.Error("failed to provision default auth user", "error", err)
	}

	protected := group.Group("/")
	protected.Use(h.authService.GinMiddleware())

	if middleware != nil {
		protected.Use(*middleware)
		group.Use(*middleware)
	}

	// Health check
	group.GET("/healthcare/", handlers.GetHealthcare)
	protected.GET("/healthcare/:name", handlers.GetServiceHealth)

	// Actions
	protected.GET("/actions/", handlers.ListActions)
	protected.GET("/actions/:slug", handlers.GetAction)
	protected.POST("/actions/:slug/execute", handlers.ExecuteAction)

	// Workers
	protected.GET("/workers/", handlers.ListWorkers)
	protected.GET("/workers/:id", handlers.GetWorker)
}
