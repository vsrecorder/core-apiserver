SHELL := /bin/bash

.PHONY: test
test:
	go mod tidy
	go test -v -cover -race ./...

.PHONY: run
run:
	go mod tidy
	go run cmd/core-apiserver/main.go

.PHONY: build
build:
	go mod tidy
	go build -o /dev/null cmd/core-apiserver/main.go

PHONY: mockgen
mockgen:
	mockgen -source=./internal/domain/repository/record.go -destination=./internal/mock/mock_repository/record.go
	mockgen -source=./internal/domain/repository/user.go -destination=./internal/mock/mock_repository/user.go
	mockgen -source=./internal/domain/repository/official_event.go -destination=./internal/mock/mock_repository/officail_event.go
	mockgen -source=./internal/domain/repository/tonamel_event.go -destination=./internal/mock/mock_repository/tonamel_event.go
	mockgen -source=./internal/domain/repository/deck.go -destination=./internal/mock/mock_repository/deck.go
	mockgen -source=./internal/domain/repository/match.go -destination=./internal/mock/mock_repository/match.go
	mockgen -source=./internal/domain/repository/game.go -destination=./internal/mock/mock_repository/game.go
	mockgen -source=./internal/domain/repository/environment.go -destination=./internal/mock/mock_repository/environment.go
	mockgen -source=./internal/domain/repository/user_stat.go -destination=./internal/mock/mock_repository/user_stat.go
	mockgen -source=./internal/domain/repository/user_stat_history.go -destination=./internal/mock/mock_repository/user_stat_history.go
	mockgen -source=./internal/domain/repository/user_stat_recent.go -destination=./internal/mock/mock_repository/user_stat_recent.go
	mockgen -source=./internal/domain/repository/opponent_deck_usage_stat.go -destination=./internal/mock/mock_repository/opponent_deck_usage_stat.go
	mockgen -source=./internal/domain/repository/weekly_deck_usage_stat.go -destination=./internal/mock/mock_repository/weekly_deck_usage_stat.go
	mockgen -source=./internal/domain/repository/standard_regulation.go -destination=./internal/mock/mock_repository/standard_regulation.go
	mockgen -source=./internal/domain/repository/badge_definition.go -destination=./internal/mock/mock_repository/badge_definition.go
	mockgen -source=./internal/domain/repository/user_badge.go -destination=./internal/mock/mock_repository/user_badge.go
	mockgen -source=./internal/domain/repository/user_streak.go -destination=./internal/mock/mock_repository/user_streak.go
	mockgen -source=./internal/domain/repository/badge_stats.go -destination=./internal/mock/mock_repository/badge_stats.go
	mockgen -source=./internal/domain/repository/designation.go -destination=./internal/mock/mock_repository/designation.go
	mockgen -source=./internal/domain/repository/designation_stats.go -destination=./internal/mock/mock_repository/designation_stats.go
	mockgen -source=./internal/domain/repository/championship_series.go -destination=./internal/mock/mock_repository/championship_series.go
	mockgen -source=./internal/domain/repository/player_ranking.go -destination=./internal/mock/mock_repository/player_ranking.go
	mockgen -source=./internal/domain/repository/notification.go -destination=./internal/mock/mock_repository/notification.go

	mockgen -source=./internal/usecase/record.go -destination=./internal/mock/mock_usecase/record.go
	mockgen -source=./internal/usecase/user.go -destination=./internal/mock/mock_usecase/user.go
	mockgen -source=./internal/usecase/official_event.go -destination=./internal/mock/mock_usecase/official_event.go
	mockgen -source=./internal/usecase/tonamel_event.go -destination=./internal/mock/mock_usecase/tonamel_event.go
	mockgen -source=./internal/usecase/deck.go -destination=./internal/mock/mock_usecase/deck.go
	mockgen -source=./internal/usecase/match.go -destination=./internal/mock/mock_usecase/match.go
	mockgen -source=./internal/usecase/game.go -destination=./internal/mock/mock_usecase/game.go
	mockgen -source=./internal/usecase/environment.go -destination=./internal/mock/mock_usecase/environment.go
	mockgen -source=./internal/usecase/user_stat.go -destination=./internal/mock/mock_usecase/user_stat.go
	mockgen -source=./internal/usecase/user_stat_history.go -destination=./internal/mock/mock_usecase/user_stat_history.go
	mockgen -source=./internal/usecase/user_stat_recent.go -destination=./internal/mock/mock_usecase/user_stat_recent.go
	mockgen -source=./internal/usecase/opponent_deck_usage_stat.go -destination=./internal/mock/mock_usecase/opponent_deck_usage_stat.go
	mockgen -source=./internal/usecase/weekly_deck_usage_stat.go -destination=./internal/mock/mock_usecase/weekly_deck_usage_stat.go
	mockgen -source=./internal/usecase/standard_regulation.go -destination=./internal/mock/mock_usecase/standard_regulation.go
	mockgen -source=./internal/usecase/badge.go -destination=./internal/mock/mock_usecase/badge.go
	mockgen -source=./internal/usecase/streak.go -destination=./internal/mock/mock_usecase/streak.go
	mockgen -source=./internal/usecase/badge_evaluation.go -destination=./internal/mock/mock_usecase/badge_evaluation.go
	mockgen -source=./internal/usecase/designation_evaluation.go -destination=./internal/mock/mock_usecase/designation_evaluation.go
	mockgen -source=./internal/usecase/designation.go -destination=./internal/mock/mock_usecase/designation.go
	mockgen -source=./internal/usecase/notification.go -destination=./internal/mock/mock_usecase/notification.go

.PHONY: image
image:
	docker build . -t vsrecorder/core-apiserver:local
	docker push       vsrecorder/core-apiserver:local

.PHONY: deploy
deploy:
	git pull
	docker compose pull
	docker compose up -d --no-deps --wait core-apiserver

.PHONY: restart
restart:
	docker compose down
	docker compose up -d

.PHONY: up
up:
	docker compose up -d

.PHONY: down
down:
	docker compose down

.PHONY: log
log:
	docker logs -f core-apiserver-core-apiserver-1
