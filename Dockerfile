FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o job-board .

FROM alpine:3.21

RUN adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /build/job-board .
COPY --from=builder /build/prompts/ ./prompts/
COPY --from=builder /build/static/ ./static/

USER appuser

ENTRYPOINT ["./job-board"]
