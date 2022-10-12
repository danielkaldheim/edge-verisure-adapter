ARG GO_VERSION=1.18
FROM golang:${GO_VERSION}-alpine AS builder

RUN apk update && apk add --no-cache git ca-certificates wget tzdata

WORKDIR /src
COPY ./src .
RUN mkdir -p data

RUN go mod download
RUN go get -v ./
RUN CGO_ENABLED=0 GOOS=linux go build -installsuffix 'static' -tags=timetzdata -o service .


FROM scratch
LABEL maintainer "Daniel Rufus Kaldheim <daniel@kaldheim.org>"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo/Europe/Oslo /usr/share/zoneinfo/Europe/Oslo
COPY --from=builder /src/service /app/service
COPY --from=builder /src/data /app/data

COPY ./testdata/defaults /app/defaults

WORKDIR /app

CMD ["./service"]