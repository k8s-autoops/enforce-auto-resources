FROM golang:1.14 AS builder
ENV GOPROXY https://goproxy.io
ENV CGO_ENABLED 0
WORKDIR /go/src/app
ADD . .
RUN go build -mod vendor -o /enforce-auto-resources

FROM alpine:3.12
COPY --from=builder /enforce-auto-resources /enforce-auto-resources
CMD ["/enforce-auto-resources"]