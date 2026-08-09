FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
COPY configs ./configs
COPY migrations ./migrations
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ingestion-worker ./cmd/ingestion-worker \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bootstrap-admin ./cmd/bootstrap-admin

FROM alpine:3.21
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=builder /out/server /out/worker /out/ingestion-worker /out/bootstrap-admin /app/
COPY --from=builder /src/configs /app/configs
COPY --from=builder /src/migrations /app/migrations
USER app
EXPOSE 8080
ENTRYPOINT ["/app/server"]
