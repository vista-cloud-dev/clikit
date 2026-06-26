# clikit — shared CLI convention layer (library module; no binary).
test:
	go test -race -cover ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

check: vet lint test
