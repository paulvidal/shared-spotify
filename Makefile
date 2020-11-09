SHELL := /bin/bash

run:
	source env.sh && rm -rf app.log && go run main.go

front:
	yarn --cwd frontend dev

mongo:
	run-rs --mongod --dbpath /usr/local/var/mongodb --keep

connect:
	mongo "mongodb://localhost:27017,localhost:27018,localhost:27019/?replicaSet=rs"