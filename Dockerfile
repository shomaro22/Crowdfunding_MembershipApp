# syntax=docker/dockerfile:1

############################
# Backend build/runtime
############################
FROM golang:1.25-alpine AS backend-build
WORKDIR /src/backend
RUN apk add --no-cache git ca-certificates
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/crowdfund-app .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.21 AS backend-runtime
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
RUN mkdir -p /app/data/public /app/data/private
COPY --from=backend-build /out/crowdfund-app .
COPY --from=backend-build /out/seed .
ENV PORT=8080
EXPOSE 8080
CMD ["./crowdfund-app"]

############################
# Frontend build/runtime
############################
FROM node:22-alpine AS frontend-build
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ ./
ENV VITE_API_BASE_URL=
RUN npm run build

FROM nginx:1.27-alpine AS frontend-runtime
COPY --from=frontend-build /src/frontend/dist /usr/share/nginx/html
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
