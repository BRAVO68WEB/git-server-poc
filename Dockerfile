# syntax=docker/dockerfile:1.6
FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/githut ./cmd/githut

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/githut /usr/local/bin/githut
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/githut"]
CMD ["serve"]
