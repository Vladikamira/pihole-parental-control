FROM golang:1.25-alpine

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .

RUN go build -o main cmd/main.go

CMD ["./main"]