FROM golang:1.19

WORKDIR /usr/src/keygen

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

EXPOSE 8000

RUN go build -o keyswarm .


ENTRYPOINT ./keyswarm