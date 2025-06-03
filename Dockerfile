FROM golang:1.24-alpine AS build

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN apk update && apk add --no-cache make
RUN make build-radio

FROM alpine:latest

COPY --from=build /build/bin/radio-style-stream /usr/bin

ENTRYPOINT ["radio-style-stream"]
