#!/usr/bin/env bash
cd test
docker-compose build
docker-compose up --wait
go run main.go
docker-compose down -v
cd ..