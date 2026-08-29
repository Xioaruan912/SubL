FROM node:22-alpine AS frontend
WORKDIR /app/webs
RUN corepack enable
COPY webs/package.json webs/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY webs/ ./
RUN pnpm build

FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/webs/dist ./webs/dist
RUN go build -tags "with_utls with_quic" -ldflags="-w -s" -o /out/ppeelink .

FROM alpine:3.22
WORKDIR /app
ENV TZ=Asia/Shanghai
RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /app/db /app/logs /app/template
COPY --from=builder /out/ppeelink /app/ppeelink
EXPOSE 8000
VOLUME ["/app/db", "/app/template", "/app/logs"]
ENTRYPOINT ["/app/ppeelink"]
