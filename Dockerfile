# --- Build stage ---------------------------------------------------------
FROM golang:1.22-alpine AS build

WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy the rest of the source and build the static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/radioapp ./cmd/server

# --- Runtime stage -------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app

WORKDIR /app
COPY --from=build /out/radioapp /app/radioapp
COPY web /app/web
COPY migrations /app/migrations

USER app
EXPOSE 8080
ENV PORT=8080

ENTRYPOINT ["/app/radioapp"]
