FROM golang:1.22.3-alpine3.19 as build
ADD ./ ./
RUN go build main.go

FROM alpine:latest
COPY --from=build /go/main /app/main
WORKDIR /app
CMD [ "./main" ]