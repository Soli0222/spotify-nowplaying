FROM golang:1.24.6-alpine3.22 AS build
ADD ./ ./
RUN go build main.go

FROM alpine:3.22.2
COPY --from=build /go/main /app/main
WORKDIR /app
USER 1001
CMD [ "./main" ]