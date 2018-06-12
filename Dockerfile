FROM alpine:3.7
LABEL maintainer="anli@spreadshirt.net"

COPY f3 /usr/bin
ENV  F3_COMMANDS="--s3-region=eu-central-1 --s3-credentials=ACCESSKEY:SECRETKEY --s3-bucket='https://my-bucket.s3.amazonaws.com'"
VOLUME ["/etc/f3"]
EXPOSE 21

CMD f3 --ftp-addr=0.0.0.0:21 $F3_COMMANDS /etc/f3/credentials.txt
