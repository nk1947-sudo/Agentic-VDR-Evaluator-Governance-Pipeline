package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/models"
)

type MockSnowflakeStaging struct {
	stagingDir string
}

func NewMockSnowflakeStaging(stagingDir string) *MockSnowflakeStaging {
	_ = os.MkdirAll(stagingDir, 0755)
	return &MockSnowflakeStaging{stagingDir: stagingDir}
}

// StageDealMetrics exports approved, sanitized deal metrics into Snowflake ingestion stage
func (sf *MockSnowflakeStaging) StageDealMetrics(
	deal *models.Deal,
	records []*models.FinancialMetricsRecord,
	risks []models.DealRisk,
) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("stage_deal_%s_%s.json", deal.ID, timestamp)
	fullPath := filepath.Join(sf.stagingDir, filename)

	payload := map[string]interface{}{
		"deal_id":         deal.ID,
		"company_name":    deal.CompanyName,
		"target_industry": deal.TargetIndustry,
		"document_hash":   deal.DocumentHash,
		"status":          deal.Status,
		"staged_at":       time.Now().UTC().Format(time.RFC3339),
		"financials":      records,
		"risks":           risks,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}

	err = os.WriteFile(fullPath, data, 0644)
	if err != nil {
		return "", err
	}

	return fullPath, nil
}
