.PHONY: test
test:
	go mod tidy
	go test -v -cover -race ./...

.PHONY: build
build:
	go mod tidy
	go build -o bin/core-apiserver cmd/core-apiserver/main.go

.PHONY: run
run:
	go mod tidy
	go run cmd/core-apiserver/main.go

.PHONY: mockgen
mockgen:
	mockgen -source=./internal/domain/repository/record.go -destination=./internal/mock/mock_repository/record.go
	mockgen -source=./internal/domain/repository/user.go -destination=./internal/mock/mock_repository/user.go
	mockgen -source=./internal/domain/repository/official_event.go -destination=./internal/mock/mock_repository/officail_event.go
	mockgen -source=./internal/domain/repository/tonamel_event.go -destination=./internal/mock/mock_repository/tonamel_event.go
	mockgen -source=./internal/domain/repository/deck.go -destination=./internal/mock/mock_repository/deck.go
	mockgen -source=./internal/domain/repository/match.go -destination=./internal/mock/mock_repository/match.go
	mockgen -source=./internal/domain/repository/game.go -destination=./internal/mock/mock_repository/game.go

	mockgen -source=./internal/usecase/record.go -destination=./internal/mock/mock_usecase/record.go
	mockgen -source=./internal/usecase/user.go -destination=./internal/mock/mock_usecase/user.go
	mockgen -source=./internal/usecase/official_event.go -destination=./internal/mock/mock_usecase/official_event.go
	mockgen -source=./internal/usecase/tonamel_event.go -destination=./internal/mock/mock_usecase/tonamel_event.go
	mockgen -source=./internal/usecase/deck.go -destination=./internal/mock/mock_usecase/deck.go
	mockgen -source=./internal/usecase/match.go -destination=./internal/mock/mock_usecase/match.go
	mockgen -source=./internal/usecase/game.go -destination=./internal/mock/mock_usecase/game.go
