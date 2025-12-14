FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod go mod download
RUN go build -o /out/githut ./cmd/githut

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/githut /usr/local/bin/githut
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s CMD wget -qO- http://localhost:8080/readyz || exit 1
ENTRYPOINT ["/usr/local/bin/githut"]
CMD ["serve"]
