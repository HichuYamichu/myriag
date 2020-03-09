FROM golang:latest AS build

WORKDIR /build

ENV GO111MODULE=on

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o myriag .

FROM scratch

WORKDIR /app

COPY --from=build /build/myriag /app/

EXPOSE 3000

CMD ["./myriag"]