# Phony targets are not real files. Declaring them avoids conflicts with files of the same name.
.PHONY: vet lint

# vet: Runs go vet to check for suspicious constructs in the code.
vet:
	@echo "Vetting code..."
	go vet ./...

# lint: Runs golangci-lint to find style issues and potential bugs.
# Requires golangci-lint to be installed: https://golangci-lint.run/usage/install/
lint:
	@echo "Linting code..."
	golangci-lint run ./...