FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/eis-customer ./cmd/eis-customer

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 1000 appuser
WORKDIR /app
COPY --from=build /out/eis-customer /app/eis-customer
COPY cert/ /app/cert/
RUN mkdir -p /app/fixtures/fz44 /app/fixtures/fz223 \
    && chown -R appuser:appuser /app/fixtures
USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/eis-customer"]
