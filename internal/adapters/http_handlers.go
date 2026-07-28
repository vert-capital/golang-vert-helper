package adapters

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/vert-capital/golang-vert-helper/internal/domain"
	"github.com/vert-capital/golang-vert-helper/internal/services"
	healthchecks "github.com/vert-capital/golang-vert-helper/pkg/health_checks"
)

// Handlers agrupa os handlers HTTP da biblioteca
type Handlers struct {
	healthService *services.HealthService
	actionService *services.ActionService
	authService   *services.AuthService
	workerPool    *healthchecks.WorkerPool
}

// NewHandlers cria os handlers com os serviços necessários
func NewHandlers(
	healthService *services.HealthService,
	actionService *services.ActionService,
	authService *services.AuthService,
	workerPool *healthchecks.WorkerPool,
) *Handlers {
	return &Handlers{
		healthService: healthService,
		actionService: actionService,
		authService:   authService,
		workerPool:    workerPool,
	}
}

type authLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthLogin autentica usuario e retorna um token JWT.
func (h *Handlers) AuthLogin(c *gin.Context) {
	var req authLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	token, user, err := h.authService.Authenticate(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch err {
		case domain.ErrInvalidCredentials, domain.ErrUserInactive:
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"token_type": "Bearer",
		"expires_in": int(h.authService.TokenTTL().Seconds()),
		"user": gin.H{
			"id":           user.ID,
			"email":        user.Email,
			"name":         user.Name,
			"is_superuser": user.IsSuperuser,
		},
	})
}

// ========== Health Check Handlers ==========

// GetHealthcare retorna o status de saúde de todos os serviços
func (h *Handlers) GetHealthcare(c *gin.Context) {
	forceRefreshRaw := c.DefaultQuery("force_refresh", "false")
	forceRefresh, parseErr := strconv.ParseBool(forceRefreshRaw)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid force_refresh query param"})
		return
	}

	if forceRefresh {
		freshResults := h.healthService.CheckAll(c.Request.Context())
		response := make(map[string]gin.H)
		for serviceName, result := range freshResults {
			response[serviceName] = gin.H{
				"status":       string(result.Status),
				"message":      result.Message,
				"last_updated": result.Timestamp,
			}
		}

		c.JSON(http.StatusOK, response)
		return
	}

	statuses, err := h.healthService.GetAllLatestStatuses(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Transformar array em mapa com service names como chaves
	response := make(map[string]gin.H)
	for _, status := range statuses {
		if status.Service == nil {
			continue // Skip se não conseguir carregar o serviço
		}
		response[status.Service.Name] = gin.H{
			"status":       string(status.Status),
			"message":      status.Message,
			"last_updated": status.CheckedAt,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetServiceHealth retorna o status de saúde de um serviço específico
func (h *Handlers) GetServiceHealth(c *gin.Context) {
	name := c.Param("name")
	forceRefreshRaw := c.DefaultQuery("force_refresh", "false")
	forceRefresh, parseErr := strconv.ParseBool(forceRefreshRaw)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid force_refresh query param"})
		return
	}

	if forceRefresh {
		result, err := h.healthService.CheckService(c.Request.Context(), name)
		if err != nil {
			if err == domain.ErrCheckerNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "no checker registered for this service"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
		return
	}

	result, err := h.healthService.GetLatestStatus(c.Request.Context(), name)
	if err != nil {
		if err == domain.ErrServiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ========== Action Handlers ==========

// ListActions retorna as actions disponíveis de um serviço
func (h *Handlers) ListActions(c *gin.Context) {
	serviceID := c.DefaultQuery("service_id", "")

	pageRaw := c.DefaultQuery("page", "1")
	pageSizeRaw := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageRaw)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page query param"})
		return
	}
	pageSize, err := strconv.Atoi(pageSizeRaw)
	if err != nil || pageSize < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page_size query param"})
		return
	}

	paginated, err := h.actionService.ListActions(c.Request.Context(), serviceID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":       paginated.Count,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": paginated.TotalPages,
		"results":     paginated.Results,
	})
}

// GetAction retorna uma action pelo slug
func (h *Handlers) GetAction(c *gin.Context) {
	slug := c.Param("slug")

	action, err := h.actionService.GetAction(c.Request.Context(), slug)
	if err != nil {
		if err == domain.ErrActionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "action not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, action)
}

// ExecuteAction executa uma action com o input do request body
func (h *Handlers) ExecuteAction(c *gin.Context) {
	slug := c.Param("slug")

	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	result, err := h.actionService.Execute(c.Request.Context(), slug, input)
	if err != nil {
		switch err {
		case domain.ErrActionNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "action not found"})
		case domain.ErrInvalidResponses:
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

// ========== Worker Handlers ==========

// ListWorkers retorna todos os workers registrados no pool
func (h *Handlers) ListWorkers(c *gin.Context) {
	if h.workerPool == nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	c.JSON(http.StatusOK, h.workerPool.GetAll())
}

// GetWorker retorna um worker específico pelo ID
func (h *Handlers) GetWorker(c *gin.Context) {
	id := c.Param("id")

	if h.workerPool == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "worker not found"})
		return
	}

	worker, ok := h.workerPool.GetByID(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "worker not found"})
		return
	}

	c.JSON(http.StatusOK, worker)
}
