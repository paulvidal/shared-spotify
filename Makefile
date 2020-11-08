SHELL := /bin/bash

run:
	source env.sh && rm -rf app.log && go run main.go

front:
	yarn --cwd frontend dev

mongo:
	mongod --config /usr/local/etc/mongod.conf

connect:
	mongo spotify