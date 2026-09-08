.PHONY: all setup seed test test-py test-go test-adversarial docker-up docker-down clean

all: test

setup:
	@echo "Setting up Python virtual environment and dependencies..."
	python -m pip install --upgrade pip
	pip install -r services/ai-engine/requirements.txt
	@echo "Setting up Go dependencies..."
	cd services/orchestrator && go mod tidy

seed:
	@echo "Generating synthetic CIM documents (Clean, Contradictory, PII-tainted)..."
	python scripts/seed_mock_data.py

test-py:
	@echo "Running Python AI Engine tests..."
	pytest services/ai-engine/tests -v

test-go:
	@echo "Running Go Orchestrator & Governance tests..."
	cd services/orchestrator && go test -v ./...

test-adversarial: seed
	@echo "Running End-to-End Adversarial Evaluation Suite..."
	pytest tests/adversarial/test_adversarial_pipeline.py -v -s

test: test-py test-go test-adversarial

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

clean:
	@echo "Cleaning temporary files and build artifacts..."
	rm -rf tests/fixtures/*.pdf
	rm -rf data/snowflake_staging/*
