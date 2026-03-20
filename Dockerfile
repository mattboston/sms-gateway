FROM --platform=$BUILDPLATFORM node:22-bookworm-slim AS web-build
WORKDIR /app/src/web

COPY src/web/package.json src/web/package-lock.json* ./
RUN npm ci

COPY src/web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS go-build
WORKDIR /app/src

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ ./
COPY --from=web-build /app/src/web/dist ./web/dist

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/sms-gateway ./cmd/sms-gateway

FROM debian:bookworm-slim AS runtime
WORKDIR /opt/sms-gateway

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

COPY --from=go-build /out/sms-gateway /usr/local/bin/sms-gateway

EXPOSE 5174

ENTRYPOINT ["sms-gateway"]
CMD ["serve"]
