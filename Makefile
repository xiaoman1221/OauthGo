# OauthGo 一键构建（前端 + 后端）
.PHONY: all frontend backend run clean

all: frontend backend

frontend:
	cd web && npm install && npm run build

backend:
	go build -o bin/oauthgo .

run: all
	./bin/oauthgo

clean:
	rm -rf web/dist bin
