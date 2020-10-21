SHELL := /bin/bash

run:
	source env.sh && go run main.go

front:
	yarn --cwd frontend dev