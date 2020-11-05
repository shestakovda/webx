all: test

test:
	@goimports -w .
	@go mod tidy
	@go test -timeout 10s -race -count 1 -cover -coverprofile=./webx.cover ./...

cover: test
	@go tool cover -html=./webx.cover
