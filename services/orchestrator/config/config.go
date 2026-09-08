package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                         string
	AIEngineURL                  string
	DBHost                       string
	DBPort                       string
	DBUser                       string
	DBPassword                   string
	DBName                       string
	MinGroundingScore            float64
	MaxDiscrepancies             int
	CustomerConcentrationLimit   float64
	ReviewTokenSecret            string
	SnowflakeStagingDir          string
}

func LoadConfig() *Config {
	port := getEnv("PORT", "8080")
	aiEngineURL := getEnv("AI_ENGINE_URL", "http://localhost:8000")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "vdr_user")
	dbPassword := getEnv("DB_PASSWORD", "vdr_password")
	dbName := getEnv("DB_NAME", "vdr_governance")
	secret := getEnv("REVIEW_TOKEN_SECRET", "vdr-governance-super-secret-key-change-in-production")
	stagingDir := getEnv("SNOWFLAKE_STAGING_DIR", "./data/snowflake_staging")

	minScoreStr := getEnv("MIN_GROUNDING_SCORE", "0.85")
	minScore, err := strconv.ParseFloat(minScoreStr, 64)
	if err != nil {
		minScore = 0.85
	}

	maxDiscStr := getEnv("MAX_DISCREPANCIES", "0")
	maxDisc, err := strconv.Atoi(maxDiscStr)
	if err != nil {
		maxDisc = 0
	}

	concLimitStr := getEnv("CUSTOMER_CONCENTRATION_LIMIT", "25.0")
	concLimit, err := strconv.ParseFloat(concLimitStr, 64)
	if err != nil {
		concLimit = 25.0
	}

	return &Config{
		Port:                       port,
		AIEngineURL:                aiEngineURL,
		DBHost:                     dbHost,
		DBPort:                     dbPort,
		DBUser:                     dbUser,
		DBPassword:                 dbPassword,
		DBName:                     dbName,
		MinGroundingScore:          minScore,
		MaxDiscrepancies:           maxDisc,
		CustomerConcentrationLimit: concLimit,
		ReviewTokenSecret:          secret,
		SnowflakeStagingDir:        stagingDir,
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
