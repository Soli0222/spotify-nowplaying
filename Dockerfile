FROM --platform=$BUILDPLATFORM golang:1.25.5-alpine3.22 AS build
ARG TARGETOS
ARG TARGETARCH
ADD ./ ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o server ./cmd/server

FROM alpine:3.23.2
COPY --from=build /go/server /app/server
WORKDIR /app
USER 1001
CMD [ "./server" ]