package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	App      AppConfig
	DB       DBConfig
	Redis    RedisConfig
	Qdrant   QdrantConfig
	LLM      LLMConfig
	JWT      JWTConfig
	KMS      KMSConfig
	Firebase FirebaseConfig
	SendGrid SendGridConfig
	GCS      GCSConfig
	Server   ServerConfig
	CORS     CORSConfig
}

// CORSConfig controls which origins the server accepts cross-origin requests from.
type CORSConfig struct {
	// AllowedOrigins is a list of origins (e.g. "https://app.example.com").
	// Use ["*"] to allow all origins (development only).
	AllowedOrigins []string
}

type AppConfig struct {
	Env     string // development, staging, production
	Name    string
	Version string
	Debug   bool
}

type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	RateLimit       int // requests per second per IP
}

type DBConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	AnonDSN         string // 独立连接用于 anon schema（最小权限）
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

type QdrantConfig struct {
	Host       string
	Port       int
	APIKey     string
	TLSEnabled bool
	Collections QdrantCollections
}

type QdrantCollections struct {
	Agents string
	Topics string
}

type LLMConfig struct {
	AnthropicAPIKey  string
	OpenAIAPIKey     string
	DeepSeekAPIKey   string
	VoyageAPIKey     string

	// 路由策略
	PrimaryProvider  string // anthropic, openai, deepseek
	FallbackProvider string

	// 模型选择
	PrimaryModel     string // claude-sonnet-4-6
	ComplexModel     string // claude-opus-4-6
	LightModel       string // claude-haiku-4-5-20251001 or deepseek-chat
	EmbeddingModel   string // voyage-3

	// 成本控制
	MaxMonthlyUSD float64

	// 缓存 TTL
	PromptCacheTTL    time.Duration // 24h
	EmbeddingCacheTTL time.Duration // 7d
	ReportCacheTTL    time.Duration // 7d
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type KMSConfig struct {
	ProjectID  string
	LocationID string
	KeyRingID  string
	KeyID      string
}

type FirebaseConfig struct {
	ProjectID       string
	CredentialsFile string
}

type SendGridConfig struct {
	APIKey    string
	FromEmail string
	FromName  string
}

type GCSConfig struct {
	ProjectID  string
	BucketName string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	// 加载 .env 文件（开发环境）
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Env:     getEnv("APP_ENV", "development"),
			Name:    "digital-twin-community",
			Version: getEnv("APP_VERSION", "0.1.0"),
			Debug:   getEnvBool("APP_DEBUG", false),
		},
		Server: ServerConfig{
			Port:            getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:     getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
			RateLimit:       getEnvInt("SERVER_RATE_LIMIT", 100),
		},
		DB: DBConfig{
			DSN:             requireEnv("DATABASE_URL"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			AnonDSN:         getEnv("ANON_DATABASE_URL", ""),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			PoolSize: getEnvInt("REDIS_POOL_SIZE", 10),
		},
		Qdrant: QdrantConfig{
			Host:       getEnv("QDRANT_HOST", "localhost"),
			Port:       getEnvInt("QDRANT_PORT", 6334),
			APIKey:     getEnv("QDRANT_API_KEY", ""),
			TLSEnabled: getEnvBool("QDRANT_TLS_ENABLED", false),
			Collections: QdrantCollections{
				Agents: getEnv("QDRANT_COLLECTION_AGENTS", "agents"),
				Topics: getEnv("QDRANT_COLLECTION_TOPICS", "topics"),
			},
		},
		LLM: LLMConfig{
			AnthropicAPIKey:   getEnv("ANTHROPIC_API_KEY", ""),
			OpenAIAPIKey:      getEnv("OPENAI_API_KEY", ""),
			DeepSeekAPIKey:    getEnv("DEEPSEEK_API_KEY", ""),
			VoyageAPIKey:      getEnv("VOYAGE_API_KEY", ""),
			PrimaryProvider:   getEnv("LLM_PRIMARY_PROVIDER", "anthropic"),
			FallbackProvider:  getEnv("LLM_FALLBACK_PROVIDER", "deepseek"),
			PrimaryModel:      getEnv("LLM_PRIMARY_MODEL", "claude-sonnet-4-6"),
			ComplexModel:      getEnv("LLM_COMPLEX_MODEL", "claude-opus-4-6"),
			LightModel:        getEnv("LLM_LIGHT_MODEL", "claude-haiku-4-5-20251001"),
			EmbeddingModel:    getEnv("LLM_EMBEDDING_MODEL", "voyage-3"),
			MaxMonthlyUSD:     getEnvFloat("LLM_MAX_MONTHLY_USD", 500.0),
			PromptCacheTTL:    getEnvDuration("LLM_PROMPT_CACHE_TTL", 24*time.Hour),
			EmbeddingCacheTTL: getEnvDuration("LLM_EMBEDDING_CACHE_TTL", 7*24*time.Hour),
			ReportCacheTTL:    getEnvDuration("LLM_REPORT_CACHE_TTL", 7*24*time.Hour),
		},
		JWT: JWTConfig{
			Secret:          requireEnv("JWT_SECRET"),
			AccessTokenTTL:  getEnvDuration("JWT_ACCESS_TTL", 24*time.Hour),
			RefreshTokenTTL: getEnvDuration("JWT_REFRESH_TTL", 30*24*time.Hour),
		},
		KMS: KMSConfig{
			ProjectID:  getEnv("GCP_PROJECT_ID", ""),
			LocationID: getEnv("KMS_LOCATION", "global"),
			KeyRingID:  getEnv("KMS_KEY_RING", "digital-twin"),
			KeyID:      getEnv("KMS_KEY_ID", "contact-encryption"),
		},
		Firebase: FirebaseConfig{
			ProjectID:       getEnv("FIREBASE_PROJECT_ID", ""),
			CredentialsFile: getEnv("FIREBASE_CREDENTIALS_FILE", ""),
		},
		SendGrid: SendGridConfig{
			APIKey:    getEnv("SENDGRID_API_KEY", ""),
			FromEmail: getEnv("SENDGRID_FROM_EMAIL", "noreply@digital-twin.community"),
			FromName:  getEnv("SENDGRID_FROM_NAME", "数字分身社区"),
		},
		GCS: GCSConfig{
			ProjectID:  getEnv("GCP_PROJECT_ID", ""),
			BucketName: getEnv("GCS_BUCKET_NAME", ""),
		},
	}

	cfg.CORS = CORSConfig{
		AllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "*"), ","),
	}

	return cfg, nil
}

// helper functions

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return v
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}
