# 启动服务
run:
	go build ./cmd/lanmei
	docker compose config
	docker compose up -d --build

rs-build:
	cd ./internal/bot/utils/lib/wcloud
	cargo build --release

# 关闭服务
down:
	docker compose down
