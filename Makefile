SHELL := /bin/bash

.PHONY: test
test:
	go mod tidy
	go test -v -cover -race ./...

# integration-test は使い捨てのPostgresコンテナを起動し、db/schema.sql を適用した上で
# リポジトリ層の統合テスト(TestIntegration*)を実行する。終了後にコンテナは破棄される。
.PHONY: integration-test
integration-test:
	docker rm -f vsrecorder-test-db 2>/dev/null || true
	docker run -d --name vsrecorder-test-db \
		-e POSTGRES_USER=vsrecorder -e POSTGRES_PASSWORD=vsrecorder -e POSTGRES_DB=vsrecorder_test \
		-e TZ=Asia/Tokyo -p 15432:5432 postgres:16-alpine
	@until docker exec vsrecorder-test-db pg_isready -U vsrecorder >/dev/null 2>&1; do sleep 1; done
	docker exec -i vsrecorder-test-db psql -q -U vsrecorder -d vsrecorder_test < db/schema.sql
	VSRECORDER_TEST_DATABASE_URL="host=localhost port=15432 user=vsrecorder password=vsrecorder dbname=vsrecorder_test sslmode=disable TimeZone=Asia/Tokyo" \
		go test -count=1 -v -run TestIntegration ./internal/infrastructure/ ; \
		status=$$?; docker rm -f vsrecorder-test-db >/dev/null; exit $$status

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
	mockgen -source=./internal/domain/repository/official_event.go -destination=./internal/mock/mock_repository/official_event.go
	mockgen -source=./internal/domain/repository/tonamel_event.go -destination=./internal/mock/mock_repository/tonamel_event.go
	mockgen -source=./internal/domain/repository/tonamel_event_store.go -destination=./internal/mock/mock_repository/tonamel_event_store.go
	mockgen -source=./internal/domain/repository/deck.go -destination=./internal/mock/mock_repository/deck.go
	mockgen -source=./internal/domain/repository/deck_code.go -destination=./internal/mock/mock_repository/deck_code.go
	mockgen -source=./internal/domain/repository/deck_asset.go -destination=./internal/mock/mock_repository/deck_asset.go
	mockgen -source=./internal/domain/repository/match.go -destination=./internal/mock/mock_repository/match.go
	mockgen -source=./internal/domain/repository/game.go -destination=./internal/mock/mock_repository/game.go
	mockgen -source=./internal/domain/repository/environment.go -destination=./internal/mock/mock_repository/environment.go
	mockgen -source=./internal/domain/repository/user_stat.go -destination=./internal/mock/mock_repository/user_stat.go
	mockgen -source=./internal/domain/repository/user_stat_history.go -destination=./internal/mock/mock_repository/user_stat_history.go
	mockgen -source=./internal/domain/repository/user_stat_recent.go -destination=./internal/mock/mock_repository/user_stat_recent.go
	mockgen -source=./internal/domain/repository/opponent_deck_usage_stat.go -destination=./internal/mock/mock_repository/opponent_deck_usage_stat.go
	mockgen -source=./internal/domain/repository/deck_usage_stat.go -destination=./internal/mock/mock_repository/deck_usage_stat.go
	mockgen -source=./internal/domain/repository/kizuna.go -destination=./internal/mock/mock_repository/kizuna.go
	mockgen -source=./internal/domain/repository/oldest_record.go -destination=./internal/mock/mock_repository/oldest_record.go
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
	mockgen -source=./internal/domain/repository/user_environment_badge.go -destination=./internal/mock/mock_repository/user_environment_badge.go
	mockgen -source=./internal/domain/repository/user_player.go -destination=./internal/mock/mock_repository/user_player.go
	mockgen -source=./internal/domain/repository/transaction.go -destination=./internal/mock/mock_repository/transaction.go
	mockgen -source=./internal/domain/repository/calendar.go -destination=./internal/mock/mock_repository/calendar.go
	mockgen -source=./internal/domain/repository/cityleague_result.go -destination=./internal/mock/mock_repository/cityleague_result.go
	mockgen -source=./internal/domain/repository/cityleague_schedule.go -destination=./internal/mock/mock_repository/cityleague_schedule.go
	mockgen -source=./internal/domain/repository/pokemon_avatar.go -destination=./internal/mock/mock_repository/pokemon_avatar.go
	mockgen -source=./internal/domain/repository/unofficial_event.go -destination=./internal/mock/mock_repository/unofficial_event.go

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
	mockgen -source=./internal/usecase/deck_usage_stat.go -destination=./internal/mock/mock_usecase/deck_usage_stat.go
	mockgen -source=./internal/usecase/kizuna.go -destination=./internal/mock/mock_usecase/kizuna.go
	mockgen -source=./internal/usecase/oldest_record.go -destination=./internal/mock/mock_usecase/oldest_record.go
	mockgen -source=./internal/usecase/weekly_deck_usage_stat.go -destination=./internal/mock/mock_usecase/weekly_deck_usage_stat.go
	mockgen -source=./internal/usecase/standard_regulation.go -destination=./internal/mock/mock_usecase/standard_regulation.go
	mockgen -source=./internal/usecase/badge.go -destination=./internal/mock/mock_usecase/badge.go
	mockgen -source=./internal/usecase/streak.go -destination=./internal/mock/mock_usecase/streak.go
	mockgen -source=./internal/usecase/badge_evaluation.go -destination=./internal/mock/mock_usecase/badge_evaluation.go
	mockgen -source=./internal/usecase/designation_evaluation.go -destination=./internal/mock/mock_usecase/designation_evaluation.go
	mockgen -source=./internal/usecase/designation.go -destination=./internal/mock/mock_usecase/designation.go
	mockgen -source=./internal/usecase/notification.go -destination=./internal/mock/mock_usecase/notification.go
	mockgen -source=./internal/usecase/environment_badge.go -destination=./internal/mock/mock_usecase/environment_badge.go
	mockgen -source=./internal/usecase/environment_badge_evaluation.go -destination=./internal/mock/mock_usecase/environment_badge_evaluation.go
	mockgen -source=./internal/usecase/cityleague_result.go -destination=./internal/mock/mock_usecase/cityleague_result.go
	mockgen -source=./internal/usecase/calendar.go -destination=./internal/mock/mock_usecase/calendar.go
	mockgen -source=./internal/usecase/championship_series.go -destination=./internal/mock/mock_usecase/championship_series.go
	mockgen -source=./internal/usecase/cityleague_schedule.go -destination=./internal/mock/mock_usecase/cityleague_schedule.go
	mockgen -source=./internal/usecase/deck_code.go -destination=./internal/mock/mock_usecase/deck_code.go
	mockgen -source=./internal/usecase/unofficial_event.go -destination=./internal/mock/mock_usecase/unofficial_event.go
	mockgen -source=./internal/usecase/user_player.go -destination=./internal/mock/mock_usecase/user_player.go

.PHONY: image
image:
	docker build . -t vsrecorder/core-apiserver:local
	docker push       vsrecorder/core-apiserver:local

.PHONY: deploy
deploy:
	git pull
	git fetch --prune
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
