SHELL := /bin/bash

run:
	source load_env.sh && rm -rf app.log && go run main.go

front:
	yarn --cwd frontend dev

mongo:
	mongod --config /usr/local/etc/mongod.conf --replSet rs

connect:
	mongo spotify