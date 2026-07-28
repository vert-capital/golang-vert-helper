package healthchecks

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/caiofariavert/golang_vert_helper/internal/domain"
)

// S3Config contém as configurações para conexão com S3/MinIO/LocalStack
type S3Config struct {
	// Region da AWS (ex: "us-east-1", "sa-east-1")
	Region string

	// AccessKeyID e SecretAccessKey são opcionais.
	// Se não fornecidos, o SDK busca automaticamente via variáveis de ambiente,
	// IAM role, arquivo ~/.aws/credentials, etc.
	AccessKeyID     string
	SecretAccessKey string

	// IsLocal indica que a conexão é com LocalStack ou MinIO.
	// Quando true, EndpointURL e UsePathStyle são obrigatórios.
	IsLocal bool

	// EndpointURL é o endereço do LocalStack ou MinIO (ex: "http://localhost:4566")
	// Necessário apenas quando IsLocal = true
	EndpointURL string
}

// S3Checker verifica a conectividade com S3/MinIO/LocalStack
// O check é considerado bem-sucedido se o cliente e o uploader forem criados com sucesso.
type S3Checker struct {
	cfg S3Config
}

// NewS3Checker cria um novo S3Checker com a configuração fornecida
func NewS3Checker(cfg S3Config) *S3Checker {
	return &S3Checker{cfg: cfg}
}

// Check tenta criar o cliente S3 e o uploader.
// Não realiza nenhuma operação de rede contra buckets.
func (c *S3Checker) Check(ctx context.Context) (*domain.HealthCheckResult, error) {
	_, _, err := c.buildClient(ctx)
	if err != nil {
		return &domain.HealthCheckResult{
			Status:    domain.HealthStatusUnhealthy,
			Message:   fmt.Sprintf("failed to initialize S3 client: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	msg := "S3 client initialized successfully"
	if c.cfg.IsLocal {
		msg = fmt.Sprintf("S3 client initialized (local endpoint: %s)", c.cfg.EndpointURL)
	}

	return &domain.HealthCheckResult{
		Status:    domain.HealthStatusHealthy,
		Message:   msg,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"region":   c.cfg.Region,
			"is_local": c.cfg.IsLocal,
		},
	}, nil
}

// buildClient monta o cliente S3 e o uploader
func (c *S3Checker) buildClient(ctx context.Context) (*s3.Client, *manager.Uploader, error) {
	opts := []func(*aws_config.LoadOptions) error{
		aws_config.WithRegion(c.cfg.Region),
	}

	// Credenciais estáticas são opcionais — se não fornecidas, o SDK usa a cadeia padrão
	// (env vars, IAM role, ~/.aws/credentials, etc.)
	if c.cfg.AccessKeyID != "" && c.cfg.SecretAccessKey != "" {
		opts = append(opts, aws_config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				c.cfg.AccessKeyID,
				c.cfg.SecretAccessKey,
				"",
			),
		))
	}

	awsCfg, err := aws_config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var client *s3.Client
	if c.cfg.IsLocal {
		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(c.cfg.EndpointURL)
			o.UsePathStyle = true // Necessário para LocalStack e MinIO
		})
	} else {
		client = s3.NewFromConfig(awsCfg)
	}

	uploader := manager.NewUploader(client)
	return client, uploader, nil
}
