# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS backend-build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/crowdfund-app .

FROM alpine:3.21 AS backend-runtime
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
RUN mkdir -p /app/data/public
COPY --from=backend-build /out/crowdfund-app .
ENV PORT=8080
EXPOSE 8080
CMD ["./crowdfund-app"]
