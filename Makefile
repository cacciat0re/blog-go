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
	migrate -path migrations -database "postgres://postgres:$$DB_PASSWORD@localhost:5432/blog_go_db?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://postgres:$$DB_PASSWORD@localhost:5432/blog_go_db?sslmode=disable" down 1