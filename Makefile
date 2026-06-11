SHELL := /bin/sh

APP := unishare
BIN := bin/$(APP)

.PHONY: run build test check docker-build compose-up compose-down

run:
	set -a; [ ! -f .env ] || . ./.env; set +a; \
	UNISHARE_USERS=$${UNISHARE_USERS:-default:test-token} \
	UNISHARE_DATA_DIR=$${UNISHARE_DATA_DIR:-/tmp/unishare-data} \
	UNISHARE_STATIC_DIR=$${UNISHARE_STATIC_DIR:-static} \
	UNISHARE_ADDR=$${UNISHARE_ADDR:-127.0.0.1:8080} \
	go run -buildvcs=false .

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" -o $(BIN) .

test:
	GOCACHE=$${GOCACHE:-/tmp/go-build} go test -buildvcs=false ./...

check:
	GOCACHE=$${GOCACHE:-/tmp/go-build} go test -buildvcs=false ./...
	python3 -m json.tool static/manifest.webmanifest >/dev/null
	node --check static/js/api.js
	node --check static/js/app.js
	node --check static/js/i18n.js
	node --check static/js/theme.js
	node --check static/js/ui.js

docker-build:
	docker compose -f docker-compose.yml -f docker-compose.build.yml build

compose-up:
	docker compose --env-file .env up -d

compose-down:
	docker compose down
