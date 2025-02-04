.PHONY: test
test:
	go test -v -cover -race ./...

.PHONY: mockgen
mockgen:
	mockgen -source=./internal/usecase/record.go -destination=./internal/mock/mock_usecase/record.go

