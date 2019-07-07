# Backend
FROM golang:1.12-alpine as backend-build
RUN apk --no-cache add ca-certificates git
WORKDIR /build/etcdv3-browser

COPY backend/go.mod ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 go test -v ./...
RUN CGO_ENABLED=0 go build

# Frontend
FROM node:12-alpine as frontend-build
WORKDIR /build/etcdv3-browser

COPY frontend/package*.json ./
RUN npm install

COPY frontend/ ./
RUN npm run lint
RUN npm run build

# Create final image
FROM alpine
WORKDIR /root
COPY --from=frontend-build /build/etcdv3-browser/dist/ dist/
COPY --from=backend-build /build/etcdv3-browser/etcdv3-browser .
EXPOSE 8081
CMD ["./etcdv3-browser"]
