FROM --platform=$BUILDPLATFORM golang:1.26.3-alpine AS builder

ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
COPY outputs/devin-proto-go/go.mod outputs/devin-proto-go/go.sum ./outputs/devin-proto-go/
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY outputs/devin-proto-go ./outputs/devin-proto-go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/cha-k ./cmd/cha-k

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /out/cha-k /app/cha-k

EXPOSE 8080

ENTRYPOINT ["/app/cha-k"]
CMD ["--config", "/app/config.yaml"]
