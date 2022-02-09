FROM golang:1.17

WORKDIR /usr/src/app

RUN apt-get update && apt-get -y install build-essential

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/app ./...

RUN make clean

CMD make cover