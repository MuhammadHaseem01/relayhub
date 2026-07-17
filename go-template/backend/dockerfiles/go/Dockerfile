FROM golang:1.24.1-alpine

WORKDIR /cargonex

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .

RUN go build -o cargonex .

EXPOSE 5000

CMD ["./cargonex"]
