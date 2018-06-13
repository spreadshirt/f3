FROM alpine:3.7
LABEL maintainer="anli@spreadshirt.net"

RUN apk update && apk add --no-cache ca-certificates

COPY f3 /usr/bin
ENV  F3_COMMANDS="--disable-cloudwatch"
ENV  S3_REGION="eu-central-1"
ENV  S3_CREDENTIAL="ACCESSKEY:SECRETKEY"
ENV  S3_BUCKET="https://my-bucket.s3.amazonaws.com"
ENV  FTP_ADDR="0.0.0.0:21"
ENV  FTP_FEATURES="ls"

VOLUME ["/etc/f3"]
EXPOSE 21

CMD f3 $F3_COMMANDS /etc/f3/credentials.txt
