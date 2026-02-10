SHELL := /bin/bash

.PHONY: lint format deps

lint:
	./scripts/lint.sh

format:
	./scripts/format.sh

deps:
	./scripts/deps.sh
