# Frontend build stage
FROM node:26.12.1-alpine AS frontend-build
WORKDIR /app/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN npm install --global "$(node -p 'require("./package.json").packageManager')"
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

# Backend build stage
FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine3.23 AS build
ARG TARGETOS
ARG TARGETARCH
ADD ./ ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o server ./cmd/server

# Final stage
FROM alpine:3.24.3
COPY --from=build /go/server /app/server
COPY --from=frontend-build /app/frontend/dist /app/frontend/dist
WORKDIR /app
USER 1001
CMD [ "./server" ]