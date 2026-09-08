package storage

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/models"
)

type Storage interface {
	SaveDeal(deal *models.Deal) error
	GetDeal(dealID string) (*models.Deal, error)
	UpdateDealStatus(dealID string, status models.DealStatus) error
	SaveFinancials(records []*models.FinancialMetricsRecord) error
	GetFinancials(dealID string) ([]*models.FinancialMetricsRecord, error)
	SaveRisks(risks []models.DealRisk) error
	GetRisks(dealID string) ([]models.DealRisk, error)
	SaveAuditLog(dealID, step, agent, action string, score float64, discCount int, hitlTriggered bool, reviewer, decision, comment string) error
	Close() error
}

// PostgresStorage handles live PostgreSQL interactions with in-memory fallback
type PostgresStorage struct {
	db       *sql.DB
	isLive   bool
	mu       sync.RWMutex
	deals    map[string]*models.Deal
	finances map[string][]*models.FinancialMetricsRecord
	risks    map[string][]models.DealRisk
	logs     []map[string]interface{}
}

func NewPostgresStorage(host, port, user, password, dbname string) (*PostgresStorage, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=3",
		host, port, user, password, dbname,
	)

	ps := &PostgresStorage{
		deals:    make(map[string]*models.Deal),
		finances: make(map[string][]*models.FinancialMetricsRecord),
		risks:    make(map[string][]models.DealRisk),
		logs:     make([]map[string]interface{}, 0),
	}

	db, err := sql.Open("postgres", connStr)
	if err == nil && db.Ping() == nil {
		log.Printf("Connected to live PostgreSQL database at %s:%s", host, port)
		ps.db = db
		ps.isLive = true
	} else {
		log.Printf("PostgreSQL live connection not available (%v). Operating in in-memory storage mode.", err)
		ps.isLive = false
	}

	return ps, nil
}

func (s *PostgresStorage) SaveDeal(deal *models.Deal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deals[deal.ID] = deal

	if s.isLive && s.db != nil {
		query := `
			INSERT INTO deals (id, company_name, target_industry, source_filename, document_hash, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET status = $6, updated_at = $8`
		_, err := s.db.Exec(query, deal.ID, deal.CompanyName, deal.TargetIndustry, deal.SourceFilename, deal.DocumentHash, string(deal.Status), deal.CreatedAt, deal.UpdatedAt)
		return err
	}
	return nil
}

func (s *PostgresStorage) GetDeal(dealID string) (*models.Deal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deal, exists := s.deals[dealID]
	if !exists {
		return nil, fmt.Errorf("deal %s not found", dealID)
	}
	return deal, nil
}

func (s *PostgresStorage) UpdateDealStatus(dealID string, status models.DealStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	deal, exists := s.deals[dealID]
	if !exists {
		return fmt.Errorf("deal %s not found", dealID)
	}
	deal.Status = status
	deal.UpdatedAt = time.Now()

	if s.isLive && s.db != nil {
		query := `UPDATE deals SET status = $1, updated_at = $2 WHERE id = $3`
		_, err := s.db.Exec(query, string(status), deal.UpdatedAt, dealID)
		return err
	}
	return nil
}

func (s *PostgresStorage) SaveFinancials(records []*models.FinancialMetricsRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rec := range records {
		s.finances[rec.DealID] = append(s.finances[rec.DealID], rec)
		if s.isLive && s.db != nil {
			query := `
				INSERT INTO deal_financials (
					deal_id, fiscal_year, period_type, currency, revenue_cents, arr_cents,
					gross_profit_cents, gross_margin_bps, ebitda_cents, adjusted_ebitda_cents,
					net_retention_rate_bps, customer_churn_bps, citation_page, citation_quote
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
			_, _ = s.db.Exec(
				query, rec.DealID, rec.FiscalYear, rec.PeriodType, rec.Currency, rec.RevenueCents, rec.ARRCents,
				rec.GrossProfitCents, rec.GrossMarginBps, rec.EBITDACents, rec.AdjustedEBITDACents,
				rec.NetRetentionRateBps, rec.CustomerChurnBps, rec.CitationPage, rec.CitationQuote,
			)
		}
	}
	return nil
}

func (s *PostgresStorage) GetFinancials(dealID string) ([]*models.FinancialMetricsRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.finances[dealID], nil
}

func (s *PostgresStorage) SaveRisks(risks []models.DealRisk) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range risks {
		s.risks[r.DealID] = append(s.risks[r.DealID], r)
		if s.isLive && s.db != nil {
			query := `
				INSERT INTO deal_risks (
					deal_id, risk_category, severity, description, concentration_pct,
					citation_page, citation_quote, grounding_confidence
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
			_, _ = s.db.Exec(
				query, r.DealID, r.RiskCategory, r.Severity, r.Description, r.ConcentrationPct,
				r.Citation.PageNumber, r.Citation.ExactQuote, r.GroundingConfidence,
			)
		}
	}
	return nil
}

func (s *PostgresStorage) GetRisks(dealID string) ([]models.DealRisk, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.risks[dealID], nil
}

func (s *PostgresStorage) SaveAuditLog(dealID, step, agent, action string, score float64, discCount int, hitlTriggered bool, reviewer, decision, comment string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	logEntry := map[string]interface{}{
		"deal_id":        dealID,
		"step":           step,
		"agent":          agent,
		"action":         action,
		"score":          score,
		"disc_count":     discCount,
		"hitl_triggered": hitlTriggered,
		"reviewer":       reviewer,
		"decision":       decision,
		"comment":        comment,
		"timestamp":      time.Now(),
	}
	s.logs = append(s.logs, logEntry)

	if s.isLive && s.db != nil {
		query := `
			INSERT INTO audit_logs (
				deal_id, step_name, agent_name, action, grounding_score,
				discrepancies_count, hitl_triggered, hitl_reviewed_by, hitl_decision, hitl_comment
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
		_, _ = s.db.Exec(query, dealID, step, agent, action, score, discCount, hitlTriggered, reviewer, decision, comment)
	}
	return nil
}

func (s *PostgresStorage) Close() error {
	if s.isLive && s.db != nil {
		return s.db.Close()
	}
	return nil
}
