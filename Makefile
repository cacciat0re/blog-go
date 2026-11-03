APP     := blog-go
CMD     := ./cmd/server
BIN     := bin/$(APP)
LDFLAGS := -s -w

.PHONY: run build build-debug fmt tidy clean migrate-up migrate-down

run:
	go run $(CMD)

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags="$(LDFLAGS)" -o $(BIN) $(CMD)

build-debug:
	go build -o $(BIN) $(CMD)

fmt:
	gofmt -w .

tidy:
	go mod tidy

clean:
	rm -rf bin

migrate-up:
	test -f .env || { echo "missing .env: expected DSN=postgres://..." >&2; exit 1; }
	set -a; . ./.env; set +a; migrate -path migrations -database "$$DSN" up

migrate-down:
	test -f .env || { echo "missing .env: expected DSN=postgres://..." >&2; exit 1; }
	set -a; . ./.env; set +a; migrate -path migrations -database "$$DSN" down 1