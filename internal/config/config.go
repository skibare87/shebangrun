package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort       string
	DatabaseURL      string
	JWTSecret        string
	StorageType      string
	S3Endpoint       string
	S3AccessKey      string
	S3SecretKey      string
	S3Bucket         string
	LocalStoragePath string
	DefaultRateLimit int
	DefaultMaxScripts int
	DefaultMaxScriptSize int64
	GitHubClientID   string
	GitHubClientSecret string
	GoogleClientID   string
	GoogleClientSecret string
}

func Load() *Config {
	return &Config{
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		DatabaseURL:      getEnv("DATABASE_URL", "root:password@tcp(mariadb:3306)/shebang?parseTime=true"),
		JWTSecret:        getEnv("JWT_SECRET", "change-me-in-production"),
		StorageType:      getEnv("STORAGE_TYPE", "s3"),
		S3Endpoint:       getEnv("S3_ENDPOINT", "minio:9000"),
		S3AccessKey:      getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:      getEnv("S3_SECRET_KEY", "minioadmin"),
		S3Bucket:         getEnv("S3_BUCKET", "scripts"),
		LocalStoragePath: getEnv("LOCAL_STORAGE_PATH", "/data/scripts"),
		DefaultRateLimit: getEnvInt("DEFAULT_RATE_LIMIT", 50),
		DefaultMaxScripts: getEnvInt("DEFAULT_MAX_SCRIPTS", 25),
		DefaultMaxScriptSize: getEnvInt64("DEFAULT_MAX_SCRIPT_SIZE", 1048576),
		GitHubClientID:   getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		GoogleClientID:   getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	}
	return defaultVal
}
