FROM golang:1.23.4-alpine

WORKDIR /app

COPY . .

RUN go mod init api && go mod tidy
RUN go build -o main .

CMD ["./main"]