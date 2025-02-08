FROM golang:1.23.4-alpine

WORKDIR /app

RUN apk add --no-cache docker-cli

COPY . .

RUN go mod init app && go mod tidy
RUN go build -o main .

CMD ["./main"]