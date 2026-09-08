-- Agentic VDR Evaluator & Governance Pipeline
-- Golden Database Schema & Audit Ledger (PostgreSQL)

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Deals Master Table
CREATE TABLE IF NOT EXISTS deals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_name VARCHAR(255) NOT NULL,
    target_industry VARCHAR(100),
    source_filename VARCHAR(255) NOT NULL,
    document_hash VARCHAR(64) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'SUBMITTED', -- SUBMITTED, PII_SANITIZED, AUDITED, AWAITING_HUMAN_REVIEW, APPROVED, COMMITTED, REJECTED
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Structured Financial Metrics (High Precision: Cents & Basis Points)
CREATE TABLE IF NOT EXISTS deal_financials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    deal_id UUID NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
    fiscal_year INT NOT NULL,
    period_type VARCHAR(20) NOT NULL DEFAULT 'FY', -- FY, LTM, NTM
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    revenue_cents BIGINT NOT NULL,
    arr_cents BIGINT,
    gross_profit_cents BIGINT,
    gross_margin_bps INT, -- 10,000 bps = 100.00%
    ebitda_cents BIGINT NOT NULL,
    adjusted_ebitda_cents BIGINT,
    net_retention_rate_bps INT, -- e.g. 11,500 bps = 115.00%
    customer_churn_bps INT,
    citation_page INT NOT NULL,
    citation_quote TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Identified Deal Risks & Qualitative Red Flags
CREATE TABLE IF NOT EXISTS deal_risks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    deal_id UUID NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
    risk_category VARCHAR(100) NOT NULL, -- CUSTOMER_CONCENTRATION, SUPPLIER_DEPENDENCY, LITIGATION, REGULATORY, KEY_PERSON
    severity VARCHAR(20) NOT NULL, -- LOW, MEDIUM, HIGH, CRITICAL
    description TEXT NOT NULL,
    concentration_pct NUMERIC(5, 2), -- e.g. 42.50%
    citation_page INT NOT NULL,
    citation_quote TEXT NOT NULL,
    grounding_confidence NUMERIC(4, 3) NOT NULL, -- 0.000 to 1.000
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. Immutable Audit Ledger (Zero Untracked Changes)
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    deal_id UUID NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
    step_name VARCHAR(100) NOT NULL,
    agent_name VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    grounding_score NUMERIC(4, 3),
    discrepancies_count INT DEFAULT 0,
    discrepancy_details JSONB,
    hitl_triggered BOOLEAN DEFAULT FALSE,
    hitl_reviewed_by VARCHAR(100),
    hitl_decision VARCHAR(50),
    hitl_comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 5. Cryptographic Review Tokens (HITL Circuit Breaker)
CREATE TABLE IF NOT EXISTS review_tokens (
    token_hash VARCHAR(64) PRIMARY KEY,
    deal_id UUID NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_used BOOLEAN DEFAULT FALSE,
    used_at TIMESTAMP WITH TIME ZONE,
    used_by VARCHAR(100)
);

CREATE INDEX IF NOT EXISTS idx_deal_financials_deal_id ON deal_financials(deal_id);
CREATE INDEX IF NOT EXISTS idx_deal_risks_deal_id ON deal_risks(deal_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_deal_id ON audit_logs(deal_id);
CREATE INDEX IF NOT EXISTS idx_deals_status ON deals(status);
