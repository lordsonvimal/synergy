#!/bin/bash

# Check if action (up/down) is provided
if [[ -z "$1" ]]; then
  echo "Usage: $0 <up|down> [count]"
  exit 1
fi

ACTION=$1  # "up" or "down"
COUNT=${2:-""} # Optional count, default is empty (runs all migrations)

migrate -database "postgres://postgres:postgres@localhost:5433/synergy?sslmode=disable" -path ./migrations $ACTION $COUNT
