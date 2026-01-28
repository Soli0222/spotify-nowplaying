# Frontend build stage
FROM node:25.4.0-alpine3.23 AS frontend-build
WORKDIR /app/frontend
RUN npm install -g pnpm
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

# Backend build stage
FROM --platform=$BUILDPLATFORM golang:1.25.6-alpine3.22 AS build
ARG TARGETOS
ARG TARGETARCH
ADD ./ ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o server ./cmd/server

# Final stage
FROM alpine:3.23.3
COPY --from=build /go/server /app/server
COPY --from=frontend-build /app/frontend/dist /app/frontend/dist
WORKDIR /app
USER 1001
CMD [ "./server" ]