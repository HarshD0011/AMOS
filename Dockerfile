# Stage 1: Build Frontend
FROM node:20-alpine AS ui-builder
WORKDIR /app/ui
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ .
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.24-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o amos ./AMOS

# Stage 3: Final Image
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=backend-builder /app/amos .
COPY --from=ui-builder /app/ui/dist ./ui/dist
ENTRYPOINT ["./amos"]