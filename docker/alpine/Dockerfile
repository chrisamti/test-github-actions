# start a builder
FROM golang:1.18.2-alpine3.15 AS builder
LABEL maintainer="Christian Juerges, christian.juerges@tx.group"
LABEL stage=builder

ENV DO_TEST=NO

RUN set -eux; \
	apk update && apk upgrade && apk add bash \
	gcc g++ \
	make \
	openssl \
	tzdata \
	git

WORKDIR /build/test
COPY . .

