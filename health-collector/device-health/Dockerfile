FROM golang:1.19.1-alpine as builder
RUN apk update
RUN apk add git
RUN apk add ca-certificates;

WORKDIR /app
COPY main.go . 
COPY go.mod . 
COPY go.sum .

RUN go mod tidy

RUN GOOS=linux GOARCH=amd64 go build  main.go

FROM alpine

COPY --from=builder /app/main .

ENTRYPOINT ["./main", "-host=test-inserver", "-rule=edgecore", "-amqp=10.108.11.66"]