FROM golang:1.19

WORKDIR /usr/src/keyswarm

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

EXPOSE 8000

WORKDIR /usr/src/keyswarm/keys-generator
RUN go build -o /usr/src/keyswarm/xkeygen .

WORKDIR /usr/src/keyswarm
RUN go build -o xkeyswarm .


ENTRYPOINT ./xkeyswarm