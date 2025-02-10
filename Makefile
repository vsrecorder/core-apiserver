.PHONY: test
test:
	go test -v -cover -race ./...

.PHONY: mockgen
mockgen:
	mockgen -source=./internal/domain/repository/record.go -destination=./internal/mock/mock_repository/record.go
	mockgen -source=./internal/usecase/record.go -destination=./internal/mock/mock_usecase/record.go

