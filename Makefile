# Makefile — Governance Computing Formal Repository
# Phase 1: The Formal Transition
#
# Recipes are placeholders for the PL/formal-methods toolchain. Real backends
# (KaTeX renderer, Lean 4, reference compiler) are wired in as they land.

SPEC_DIR      := spec
MECH_DIR      := mechanization
COMPILER_DIR  := compiler
VALIDATION_DIR:= validation
EXPORT_DIR    := export
SPEC_FILES    := $(wildcard $(SPEC_DIR)/D1.*.md)

.DEFAULT_GOAL := help
.PHONY: help spec verify test-compiler cnf-export seal verify-seal validate clean

help: ## Show available targets
	@echo "Governance Computing — formal repository"
	@echo "Targets:"
	@echo "  make spec           Validate/render the D1 specification documents"
	@echo "  make verify         Run Lean 4 mechanization (placeholder)"
	@echo "  make test-compiler  Run the reference-compiler (Go) test suite"
	@echo "  make cnf-export     Dump the store in Canonical Normal Form to $(EXPORT_DIR)/dump.cnf"
	@echo "  make seal           Ed25519-sign the CNF export ($(EXPORT_DIR)/dump.cnf.sig)"
	@echo "  make verify-seal    Verify the CNF export seal (AUTHENTIC/TAMPERED)"
	@echo "  make validate       Run reproducibility + agreement harnesses (placeholder)"
	@echo "  make clean          Remove build artifacts"

spec: ## Validate and render the specification documents
	@echo "[spec] validating $(words $(SPEC_FILES)) document(s) in $(SPEC_DIR)/ ..."
	@for f in $(SPEC_FILES); do echo "  - $$f"; test -s "$$f" || { echo "EMPTY: $$f"; exit 1; }; done
	@echo "[spec] OK (LaTeX/KaTeX rendering backend: TODO)"

verify: ## Run the Lean 4 mechanization over the proof obligations (D1.5)
	@echo "[verify] Lean 4 mechanization — placeholder"
	@echo "[verify] discharging obligations T1..T8, C1 from $(SPEC_DIR)/D1.5-Proof-Obligations.md"
	@echo "[verify] no Lean toolchain wired yet; exiting 0 (all obligations remain conjectures)"

test-compiler: ## Run the reference-compiler (Go) test suite
	@echo "[test-compiler] go test ./... in $(COMPILER_DIR)/ ..."
	cd $(COMPILER_DIR) && go vet ./... && go test ./...

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
	@echo "[validate] reproducibility + Fleiss' kappa / verdict-agreement — placeholder"
	@echo "[validate] no harness in $(VALIDATION_DIR)/ yet; exiting 0"

clean: ## Remove build artifacts
	@echo "[clean] nothing to remove (no build artifacts yet)"
