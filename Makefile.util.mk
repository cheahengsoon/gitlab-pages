# Dockerfile https://gitlab.com/gitlab-org/gitlab-build-images/-/blob/master/Dockerfile.golangci-lint-alpine
GOLANGCI_LINT_IMAGE := golangci/golangci-lint:$(GOLANGCI_LINT_VERSION)

.PHONY: lint lint-docker test race acceptance bench cover list deps-check deps-download

OUT_FORMAT ?= colored-line-number
LINT_FLAGS ?=  $(if $V,-v)

lint: .GOPATH/.ok bin/golangci-lint
	$Q ./bin/golangci-lint run ./... --out-format $(OUT_FORMAT) $(LINT_FLAGS)

lint-docker:
	docker run -v $(PWD):/app -w /app $(GOLANGCI_LINT_IMAGE) \
	sh -c "golangci-lint run ./... --out-format $(OUT_FORMAT) $(LINT_FLAGS)"

test: .GOPATH/.ok gitlab-pages
	go test $(if $V,-v) $(allpackages)

race: .GOPATH/.ok gitlab-pages
	CGO_ENABLED=1 go test -race $(if $V,-v) $(allpackages)

acceptance: .GOPATH/.ok gitlab-pages
	go test $(if $V,-v) $(IMPORT_PATH)

bench: .GOPATH/.ok gitlab-pages
	go test -bench=. -run=^$$ $(allpackages)

# The acceptance tests cannot count for coverage
cover: bin/gocovmerge .GOPATH/.ok gitlab-pages
	@echo "NOTE: make cover does not exit 1 on failure, don't use it to check for tests success!"
	$Q rm -f .GOPATH/cover/*.out .GOPATH/cover/all.merged
	$(if $V,@echo "-- go test -coverpkg=./... -coverprofile=.GOPATH/cover/... ./...")
	@for MOD in $(allpackages); do \
		go test \
			-short \
			-coverpkg=`echo $(allpackages)|tr " " ","` \
			-coverprofile=.GOPATH/cover/unit-`echo $$MOD|tr "/" "_"`.out \
			$$MOD 2>&1 | grep -v "no packages being tested depend on"; \
	done
	$Q ./bin/gocovmerge .GOPATH/cover/*.out > .GOPATH/cover/all.merged
	$Q go tool cover -html .GOPATH/cover/all.merged -o coverage.html
	@echo ""
	@echo "=====> Total test coverage: <====="
	@echo ""
	$Q go tool cover -func .GOPATH/cover/all.merged

list: .GOPATH/.ok
	@echo $(allpackages)

deps-check: .GOPATH/.ok
	go mod tidy
	@if git diff --color=always --exit-code -- go.mod go.sum; then \
		echo "go.mod and go.sum are ok"; \
	else \
    echo ""; \
		echo "go.mod and go.sum are modified, please commit them";\
		exit 1; \
  fi;

deps-download: .GOPATH/.ok
	go mod download
