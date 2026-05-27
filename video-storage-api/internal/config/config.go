package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Auth      AuthConfig
	Storage   StorageConfig
	Streaming StreamingConfig
	HLS       HLSConfig
	Redis     RedisConfig
	Jobs      JobsConfig
	Cleanup   CleanupConfig
	CORS      CORSConfig
	App       AppConfig
}

type CORSConfig struct {
	// AllowedOrigins is an exact-match list of Origin header values
	// the browser may use to call this API (e.g. the admin dashboard).
	AllowedOrigins []string
	// AllowCredentials toggles Access-Control-Allow-Credentials.
	// Always paired with a concrete Origin (never `*`).
	AllowCredentials bool
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	MaxConns int32
}

type AuthConfig struct {
	JWTSecret         string
	JWTExpiresIn      time.Duration
	LoginMaxAttempts  int
	LoginLockDuration time.Duration
}

type StorageConfig struct {
	VideoPath         string
	ThumbnailPath     string
	HLSPath           string
	MaxVideoBytes     int64
	MaxThumbnailBytes int64
}

type HLSConfig struct {
	SegmentSeconds int
	X264Preset     string
	ConvertTimeout time.Duration
}

type StreamingConfig struct {
	SigningSecret  string
	URLExpiration  time.Duration
	AllowedDomains string // raw env value; parsed by the streaming module
	PublicBaseURL  string // e.g. "https://api.example.com"; "" → derive from request
	BindIP         bool
	BindUserAgent  bool
}

type RedisConfig struct {
	Enabled  bool
	Addr     string
	Password string
	DB       int
	Prefix   string
	TTL      time.Duration
	TLS      bool
}

type JobsConfig struct {
	WorkerConcurrency int
	MaxRetries        int
}

type CleanupConfig struct {
	Enabled        bool
	Interval       time.Duration
	TempHLSMaxAge  time.Duration
	LogMaxAge      time.Duration
	ArtifactMaxAge time.Duration
}

type AppConfig struct {
	Env string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	maxConns, err := strconv.Atoi(getEnv("DB_MAX_CONNS", "10"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_CONNS: %w", err)
	}

	jwtExpires, err := time.ParseDuration(getEnv("JWT_EXPIRES_IN", "24h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRES_IN: %w", err)
	}

	maxAttempts, err := strconv.Atoi(getEnv("LOGIN_MAX_ATTEMPTS", "5"))
	if err != nil {
		return nil, fmt.Errorf("invalid LOGIN_MAX_ATTEMPTS: %w", err)
	}
	if maxAttempts < 1 {
		return nil, fmt.Errorf("LOGIN_MAX_ATTEMPTS must be >= 1, got %d", maxAttempts)
	}

	lockMinutes, err := strconv.Atoi(getEnv("LOGIN_LOCK_DURATION_MINUTES", "15"))
	if err != nil {
		return nil, fmt.Errorf("invalid LOGIN_LOCK_DURATION_MINUTES: %w", err)
	}
	if lockMinutes < 1 {
		return nil, fmt.Errorf("LOGIN_LOCK_DURATION_MINUTES must be >= 1, got %d", lockMinutes)
	}

	env := getEnv("APP_ENV", "development")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if env != "development" && len(jwtSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 chars in non-development environments")
	}

	maxVideoMB, err := strconv.Atoi(getEnv("MAX_VIDEO_UPLOAD_SIZE_MB", "2048"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_VIDEO_UPLOAD_SIZE_MB: %w", err)
	}
	if maxVideoMB < 1 {
		return nil, fmt.Errorf("MAX_VIDEO_UPLOAD_SIZE_MB must be >= 1, got %d", maxVideoMB)
	}

	maxThumbMB, err := strconv.Atoi(getEnv("MAX_THUMBNAIL_UPLOAD_SIZE_MB", "10"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_THUMBNAIL_UPLOAD_SIZE_MB: %w", err)
	}
	if maxThumbMB < 1 {
		return nil, fmt.Errorf("MAX_THUMBNAIL_UPLOAD_SIZE_MB must be >= 1, got %d", maxThumbMB)
	}

	streamSecret := os.Getenv("STREAM_SIGNING_SECRET")
	if streamSecret == "" {
		return nil, fmt.Errorf("STREAM_SIGNING_SECRET is required")
	}
	if env != "development" && len(streamSecret) < 32 {
		return nil, fmt.Errorf("STREAM_SIGNING_SECRET must be at least 32 chars in non-development environments")
	}

	allowedStreamDomains := getEnv("ALLOWED_STREAM_DOMAINS", "")
	if env != "development" {
		if strings.TrimSpace(allowedStreamDomains) == "" || strings.TrimSpace(allowedStreamDomains) == "*" {
			return nil, fmt.Errorf("ALLOWED_STREAM_DOMAINS must list explicit domains in production")
		}
	}

	streamExpMin, err := strconv.Atoi(getEnv("STREAM_URL_EXPIRATION_MINUTES", "60"))
	if err != nil {
		return nil, fmt.Errorf("invalid STREAM_URL_EXPIRATION_MINUTES: %w", err)
	}
	if streamExpMin < 1 {
		return nil, fmt.Errorf("STREAM_URL_EXPIRATION_MINUTES must be >= 1, got %d", streamExpMin)
	}
	if env != "development" && streamExpMin > 15 {
		return nil, fmt.Errorf("STREAM_URL_EXPIRATION_MINUTES must be <= 15 in production")
	}

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}
	redisTTLSeconds, err := strconv.Atoi(getEnv("REDIS_CACHE_TTL_SECONDS", "300"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_CACHE_TTL_SECONDS: %w", err)
	}
	if redisTTLSeconds < 1 {
		return nil, fmt.Errorf("REDIS_CACHE_TTL_SECONDS must be >= 1, got %d", redisTTLSeconds)
	}
	workerConcurrency, err := strconv.Atoi(getEnv("JOB_WORKER_CONCURRENCY", "2"))
	if err != nil {
		return nil, fmt.Errorf("invalid JOB_WORKER_CONCURRENCY: %w", err)
	}
	if workerConcurrency < 1 {
		return nil, fmt.Errorf("JOB_WORKER_CONCURRENCY must be >= 1, got %d", workerConcurrency)
	}
	jobMaxRetries, err := strconv.Atoi(getEnv("JOB_MAX_RETRIES", "3"))
	if err != nil {
		return nil, fmt.Errorf("invalid JOB_MAX_RETRIES: %w", err)
	}
	if jobMaxRetries < 0 {
		return nil, fmt.Errorf("JOB_MAX_RETRIES must be >= 0, got %d", jobMaxRetries)
	}
	hlsSegmentSeconds, err := strconv.Atoi(getEnv("HLS_SEGMENT_SECONDS", "4"))
	if err != nil {
		return nil, fmt.Errorf("invalid HLS_SEGMENT_SECONDS: %w", err)
	}
	if hlsSegmentSeconds < 4 || hlsSegmentSeconds > 6 {
		return nil, fmt.Errorf("HLS_SEGMENT_SECONDS must be between 4 and 6, got %d", hlsSegmentSeconds)
	}

	allowedCORSOrigins := parseCSV(os.Getenv("ALLOWED_CORS_ORIGINS"))
	if env != "development" {
		if len(allowedCORSOrigins) == 0 {
			return nil, fmt.Errorf("ALLOWED_CORS_ORIGINS must list explicit origins in production")
		}
		for _, origin := range allowedCORSOrigins {
			if origin == "*" {
				return nil, fmt.Errorf("ALLOWED_CORS_ORIGINS cannot contain * in production")
			}
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("API_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "video_user"),
			Password: getEnv("DB_PASSWORD", "video_password"),
			Name:     getEnv("DB_NAME", "video_storage"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			MaxConns: int32(maxConns),
		},
		Auth: AuthConfig{
			JWTSecret:         jwtSecret,
			JWTExpiresIn:      jwtExpires,
			LoginMaxAttempts:  maxAttempts,
			LoginLockDuration: time.Duration(lockMinutes) * time.Minute,
		},
		Storage: StorageConfig{
			VideoPath:         getEnv("VIDEO_STORAGE_PATH", "storage/private/videos"),
			ThumbnailPath:     getEnv("THUMBNAIL_STORAGE_PATH", "storage/private/thumbnails"),
			HLSPath:           getEnv("HLS_STORAGE_PATH", "storage/private/hls"),
			MaxVideoBytes:     int64(maxVideoMB) * 1024 * 1024,
			MaxThumbnailBytes: int64(maxThumbMB) * 1024 * 1024,
		},
		Streaming: StreamingConfig{
			SigningSecret:  streamSecret,
			URLExpiration:  time.Duration(streamExpMin) * time.Minute,
			AllowedDomains: allowedStreamDomains,
			PublicBaseURL:  getEnv("PUBLIC_BASE_URL", ""),
			BindIP:         parseBool(getEnv("STREAM_BIND_IP", "false")),
			BindUserAgent:  parseBool(getEnv("STREAM_BIND_USER_AGENT", "false")),
		},
		HLS: HLSConfig{
			SegmentSeconds: hlsSegmentSeconds,
			X264Preset:     getEnv("HLS_X264_PRESET", "veryfast"),
			ConvertTimeout: parseDurationEnv("HLS_CONVERT_TIMEOUT", 60*time.Minute),
		},
		Redis: RedisConfig{
			Enabled:  parseBool(getEnv("REDIS_ENABLED", "false")),
			Addr:     getEnv("REDIS_ADDR", "redis:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
			Prefix:   getEnv("REDIS_KEY_PREFIX", "web_stream"),
			TTL:      time.Duration(redisTTLSeconds) * time.Second,
			TLS:      parseBool(getEnv("REDIS_TLS", "false")),
		},
		Jobs: JobsConfig{
			WorkerConcurrency: workerConcurrency,
			MaxRetries:        jobMaxRetries,
		},
		Cleanup: CleanupConfig{
			Enabled:        parseBool(getEnv("CLEANUP_ENABLED", "true")),
			Interval:       parseDurationEnv("CLEANUP_INTERVAL", 15*time.Minute),
			TempHLSMaxAge:  parseDurationEnv("CLEANUP_TEMP_HLS_MAX_AGE", 24*time.Hour),
			LogMaxAge:      parseDurationEnv("CLEANUP_LOG_MAX_AGE", 7*24*time.Hour),
			ArtifactMaxAge: parseDurationEnv("CLEANUP_ARTIFACT_MAX_AGE", 7*24*time.Hour),
		},
		CORS: CORSConfig{
			AllowedOrigins:   allowedCORSOrigins,
			AllowCredentials: true,
		},
		App: AppConfig{
			Env: env,
		},
	}

	return cfg, nil
}

// LoadDatabase loads only the database settings. It is intended for small
// maintenance commands that should not be blocked by unrelated API settings.
func LoadDatabase() (DatabaseConfig, error) {
	_ = godotenv.Load()

	maxConns, err := strconv.Atoi(getEnv("DB_MAX_CONNS", "10"))
	if err != nil {
		return DatabaseConfig{}, fmt.Errorf("invalid DB_MAX_CONNS: %w", err)
	}

	return DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "video_user"),
		Password: getEnv("DB_PASSWORD", "video_password"),
		Name:     getEnv("DB_NAME", "video_storage"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
		MaxConns: int32(maxConns),
	}, nil
}

// parseCSV trims and splits a comma-separated env value, dropping empties.
func parseCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&pool_max_conns=%d",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode, d.MaxConns,
	)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func parseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
