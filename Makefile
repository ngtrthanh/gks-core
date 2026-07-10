# Makefile — Governance Computing Formal Repository
# Phase 1: The Formal Transition
#
# Targets wrap the real toolchain: `verify` runs the Lean 4 mechanization
# (mathlib-free) via lake; `validate`/`ingest` run the Go harnesses against the
# live DB. See CI (`.github/workflows/lean.yml`) for the authoritative Lean build.

SPEC_DIR      := spec
MECH_DIR      := mechanization
COMPILER_DIR  := compiler
VALIDATION_DIR:= validation
EXPORT_DIR    := export
SPEC_FILES    := $(wildcard $(SPEC_DIR)/D1.*.md)

.DEFAULT_GOAL := help
.PHONY: help spec verify test-compiler replay-d8 cnf-export seal verify-seal validate ingest ingest-apply clean

help: ## Show available targets
	@echo "Governance Computing — formal repository"
	@echo "Targets:"
	@echo "  make spec           Validate/render the D1 specification documents"
	@echo "  make verify         Run the Lean 4 mechanization (lake build; models, see spec/D1.5)"
	@echo "  make test-compiler  Run the reference-compiler (Go) test suite"
	@echo "  make replay-d8      Replay the D8 benchmark traces through the persisted E-layer"
	@echo "  make cnf-export     Dump the store in Canonical Normal Form to $(EXPORT_DIR)/dump.cnf"
	@echo "  make seal           Ed25519-sign the CNF export ($(EXPORT_DIR)/dump.cnf.sig)"
	@echo "  make verify-seal    Verify the CNF export seal (AUTHENTIC/TAMPERED)"
	@echo "  make validate       Run the agreement harness on fixtures (first-party; see PROGRESS 8.2)"
	@echo "  make ingest         Continuous-ingestion control plane (dry-run)"
	@echo "  make clean          Remove build artifacts"

spec: ## Validate and render the specification documents
	@echo "[spec] validating $(words $(SPEC_FILES)) document(s) in $(SPEC_DIR)/ ..."
	@for f in $(SPEC_FILES); do echo "  - $$f"; test -s "$$f" || { echo "EMPTY: $$f"; exit 1; }; done
	@echo "[spec] OK (LaTeX/KaTeX rendering backend: TODO)"

verify: ## Run the Lean 4 mechanization over the proof obligations (D1.5)
	@if command -v lake >/dev/null 2>&1; then \
	  echo "[verify] lake build in $(MECH_DIR)/ ..."; \
	  cd $(MECH_DIR) && lake build; \
	else \
	  echo "[verify] Lean toolchain (lake) not found on PATH."; \
	  echo "[verify] The mechanization CI-compiles (Lean 4.31.0, zero sorry) but proves"; \
	  echo "[verify]   SIMPLIFIED MODELS, not the D1.5 theorems as stated — see spec/D1.5"; \
	  echo "[verify]   (T2/T5 definitional; T3/T6/T7/T8 model-lemmas; T1 scoped)."; \
	  echo "[verify] Install elan+Lean, then 'make verify' compiles them (lake build)."; \
	fi

test-compiler: ## Run the reference-compiler (Go) test suite
	@echo "[test-compiler] go test ./... in $(COMPILER_DIR)/ ..."
	cd $(COMPILER_DIR) && go vet ./... && go test ./...

replay-d8: ## Replay the D8 event traces through Ê -> persisted verdicts (WP-3)
	@echo "[replay-d8] world_event -> e_machine/transition_log/verdict"
	cd $(COMPILER_DIR) && go run ./cmd/replay_d8

cnf-export: ## Dump the store in Canonical Normal Form (deterministic, I8)
	@mkdir -p $(EXPORT_DIR)
	@echo "[cnf-export] canonicalizing kernel_instance -> $(EXPORT_DIR)/dump.cnf"
	cd $(COMPILER_DIR) && go run ./cmd/cnf_export ../$(EXPORT_DIR)/dump.cnf

seal: ## Ed25519-sign the CNF export (detached signature)
	@echo "[seal] signing $(EXPORT_DIR)/dump.cnf -> $(EXPORT_DIR)/dump.cnf.sig"
	cd $(COMPILER_DIR) && go run ./cmd/seal_export ../$(EXPORT_DIR)/dump.cnf ../$(EXPORT_DIR)/dump.cnf.sig

verify-seal: ## Verify the CNF export seal (AUTHENTIC/TAMPERED)
	cd $(COMPILER_DIR) && go run ./cmd/verify_seal ../$(EXPORT_DIR)/dump.cnf ../$(EXPORT_DIR)/dump.cnf.sig ../$(EXPORT_DIR)/ed25519_key.pub

validate: ## Run reproducibility and inter-compiler agreement harnesses
	@echo "[validate] Fleiss' kappa / verdict-agreement (floors: kappa>=0.70, VA>=0.90)"
	cd $(COMPILER_DIR) && go run ./cmd/validate ../$(VALIDATION_DIR)/testdata

ingest: ## Continuous-ingestion control plane: report NEW/CHANGED/UP-TO-DATE (dry-run)
	@echo "[ingest] dry-run over data/corpora.json (no writes)"
	cd $(COMPILER_DIR) && go run ./cmd/ingest_run

ingest-apply: ## Ingest NEW/CHANGED corpora, skip UP-TO-DATE (idempotent), ledger each run
	cd $(COMPILER_DIR) && go run ./cmd/ingest_run --apply

clean: ## Remove build artifacts
	@echo "[clean] nothing to remove (no build artifacts yet)"
