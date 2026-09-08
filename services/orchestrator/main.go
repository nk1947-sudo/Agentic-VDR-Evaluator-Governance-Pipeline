package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/api"
	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/config"
	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/statemachine"
	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/storage"
	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/validator"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting Agentic VDR Evaluator & Governance Pipeline Orchestrator on port %s", cfg.Port)

	// 1. Initialize Storage
	store, err := storage.NewPostgresStorage(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	snowflake := storage.NewMockSnowflakeStaging(cfg.SnowflakeStagingDir)

	// 2. Initialize Governance & State Machine
	tokenMgr := statemachine.NewTokenManager(cfg.ReviewTokenSecret)
	workflowMachine := statemachine.NewWorkflowMachine(tokenMgr)
	evaluator := statemachine.NewGovernanceEvaluator(
		cfg.MinGroundingScore,
		cfg.MaxDiscrepancies,
		cfg.CustomerConcentrationLimit,
	)

	// 3. Initialize Validators
	schemaVal := validator.NewSchemaValidator()
	mathRec := validator.NewMathReconciler()

	// 4. Initialize Handler & Router
	dealHandler := api.NewDealHandler(
		cfg, store, tokenMgr, workflowMachine, evaluator, schemaVal, mathRec, snowflake,
	)
	router := api.SetupRouter(dealHandler)

	addr := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
	log.Printf("Listening at http://%s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server exited: %v", err)
	}
}
