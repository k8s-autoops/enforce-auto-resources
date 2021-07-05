FROM golang:1.14 AS builder
ENV GOPROXY https://goproxy.io
ENV CGO_ENABLED 0
WORKDIR /go/src/app
ADD . .
RUN go build -mod vendor -o /template-autoops-admission

FROM alpine:3.12
COPY --from=builder /template-autoops-admission /template-autoops-admission
CMD ["/template-autoops-admission"]