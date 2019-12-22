# step 1: build
FROM golang:1.13 AS build

RUN groupadd -g 10001 myapp \
    && useradd -u 10001 -g myapp myapp

RUN mkdir /go-app
WORKDIR /go-app
COPY go.mod go.sum ./

RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/go-app

# -----------------------------------------------------------------------------
# step 2: exec
FROM alpine:3.10

# add zoneinfo(JST)
RUN apk --no-cache add tzdata && \
    cp /usr/share/zoneinfo/Asia/Tokyo /etc/localtime

COPY --from=build /go/bin/go-app /go/bin/go-app
COPY --from=build /etc/group /etc/group
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

EXPOSE 8080

USER myapp

ENTRYPOINT ["/go/bin/go-app"]