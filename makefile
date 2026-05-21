PKG_PATH=./pkg

.PHONY: fmt
fmt:
	gofumpt -l -w -extra .

.PHONY: tidy
tidy:
	cd $(PKG_PATH) && go mod tidy

.PHONY: update
update:
	cd $(PKG_PATH) && go get -u ./...

.PHONY: lint
lint: fmt
	cd $(PKG_PATH) && golangci-lint run ./... --fix
