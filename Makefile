# 启动服务
run:
	docker compose config
	docker-compose up -d
	go mod tidy
	go run ./

rs-build:
	cd ./bot/utils/lib/wcloud
	cargo build --release

# 关闭服务
down:
	docker-compose down