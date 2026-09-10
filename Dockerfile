# syntax=docker/dockerfile:1.7

FROM golang:1.26.0-bookworm AS build

ARG FLARE_VERSION=dev
WORKDIR /source

COPY go.mod go.sum ./
RUN go mod download

COPY server/ server/

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${FLARE_VERSION}" \
    -o /out/flare \
    ./server/backend/cmd/flare


FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app
COPY --from=build --chown=nonroot:nonroot /out/flare /app/flare

USER nonroot:nonroot
EXPOSE 8080

HEALTHCHECK \
    --interval=30s \
    --timeout=5s \
    --start-period=10s \
    --retries=3 \
    CMD ["/app/flare", "--healthcheck"]

ENTRYPOINT ["/app/flare"]
