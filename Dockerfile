# Backend
FROM golang:1-alpine AS backend-build
RUN apk --no-cache add ca-certificates git
WORKDIR /build/etcdv3-browser

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 go test -v ./...
RUN CGO_ENABLED=0 go build -ldflags="-s -w"

# Frontend
FROM node:22-alpine AS frontend-build
WORKDIR /build/etcdv3-browser

COPY frontend/package.json frontend/package-lock.json ./
RUN npm install

COPY frontend/ ./
RUN npm run lint
RUN npm run build

# Create final image
FROM alpine
WORKDIR /root
COPY --from=frontend-build /build/etcdv3-browser/dist/ dist/
COPY --from=backend-build /build/etcdv3-browser/etcdv3-browser .
COPY backend/templates/ ./templates/
EXPOSE 8081
CMD ["./etcdv3-browser"]
