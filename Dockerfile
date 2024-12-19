FROM golang:1.23.4 as builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o explore-server /app/cmd/main.go 

FROM builder
COPY --from=builder /app/server /server
EXPOSE 8080
ENTRYPOINT ["/bin/sh","-c","./explore-server"]