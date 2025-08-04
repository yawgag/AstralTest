FROM golang:1.24.2

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o wcs ./cmd/main.go

CMD ["./wcs"]