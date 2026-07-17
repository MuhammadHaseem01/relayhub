FROM golang:1.24.1-alpine

WORKDIR /migration

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .

ENTRYPOINT ["go", "run", "./cmd/migrate"]
