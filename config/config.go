package config

import (
	"os"

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
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
