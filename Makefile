# 启动服务
run:
	docker compose config
	docker-compose up -d
	go mod tidy
	go run ./

# 关闭服务
down:
	docker-compose down