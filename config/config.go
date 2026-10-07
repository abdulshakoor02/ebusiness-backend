package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort         string
	ServerEnv          string
	MongoURI           string
	DBName             string
	JWTSecret          string
	JWTExpiration      string
	LogLevel           string
	ServiceTenantName  string
	SuperAdminName     string
	SuperAdminEmail    string
	SuperAdminPassword string
	// AI Configuration for Chutes provider
	AIURL             string
	AIAPIKey          string
	AIModel           string
	MaxImportFileSize int64 // bytes, default 10MB
	MaxImportRows     int64 // max rows per import, default 10000

	// Facebook OAuth Integration
	FacebookAppID           string
	FacebookAppSecret       string
	FacebookRedirectURI     string
	FacebookAPIVersion      string
	FacebookScopes          []string
	FacebookPostLoginURI    string
	FacebookStateTTL        time.Duration
	FacebookWebhookVerify   string
	FacebookFrontendOrigin  string
}

func LoadConfig() *Config {
	// Try loading .env file; ignore error if file not found (e.g. in production)
	_ = godotenv.Load()

	// Handle test env overrides if needed
	if os.Getenv("SERVER_ENV") == "test" {
		_ = godotenv.Overload(".env.test")
	}

	return &Config{
		ServerPort:         getEnv("SERVER_PORT", "3000"),
		ServerEnv:          getEnv("SERVER_ENV", "development"),
		MongoURI:           getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:             getEnv("DB_NAME", "crm_db"),
		JWTSecret:          getEnv("JWT_SECRET", "secret"),
		JWTExpiration:      getEnv("JWT_EXPIRATION", "24h"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		ServiceTenantName:  getEnv("SERVICE_TENANT_NAME", "System Provider"),
		SuperAdminName:     getEnv("SUPER_ADMIN_NAME", "Super Admin"),
		SuperAdminEmail:    getEnv("SUPER_ADMIN_EMAIL", "superadmin@example.com"),
		SuperAdminPassword: getEnv("SUPER_ADMIN_PASSWORD", "superadmin123"),
		AIURL:              getEnv("AI_URL", ""),
		AIAPIKey:           getEnv("AI_API_KEY", ""),
		AIModel:            getEnv("AI_MODEL", ""),
		MaxImportFileSize:  getEnvInt64("MAX_IMPORT_FILE_SIZE", 10*1024*1024),
		MaxImportRows:      getEnvInt64("MAX_IMPORT_ROWS", 10000),
		FacebookAppID:        getEnv("FACEBOOK_APP_ID", ""),
		FacebookAppSecret:    getEnv("FACEBOOK_APP_SECRET", ""),
		FacebookRedirectURI:  getEnv("FACEBOOK_REDIRECT_URI", ""),
		FacebookAPIVersion:   getEnv("FACEBOOK_API_VERSION", "v21.0"),
		FacebookScopes:       getEnvSlice("FACEBOOK_SCOPES", []string{"ads_read", "business_management", "pages_show_list", "public_profile"}),
		FacebookPostLoginURI: getEnv("FACEBOOK_POST_LOGIN_URI", "http://localhost:3000/dashboard/company-info"),
		FacebookStateTTL:     getEnvDuration("FACEBOOK_STATE_TTL", "10m"),
		FacebookWebhookVerify:  getEnv("FACEBOOK_WEBHOOK_VERIFY_TOKEN", ""),
		FacebookFrontendOrigin: getEnv("FACEBOOK_FRONTEND_ORIGIN", "http://localhost:3000"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvSlice(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func getEnvDuration(key, fallback string) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		value = fallback
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 10 * time.Minute
	}
	return d
}
