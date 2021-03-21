SHELL := /bin/bash

run:
	source load_env.sh && rm -rf app.log && go run main.go

front:
	yarn --cwd frontend dev

mongo:
	mongod --config /usr/local/etc/mongod.conf --bind_ip_all

connect:
	mongo spotify

docker-build:
	docker build -t shared-spotify .

docker-run:
	docker run -p 8080:8080 --env-file .docker.env shared-spotify
