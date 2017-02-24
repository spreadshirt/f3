FROM debian:latest
MAINTAINER Andreas Linz <anli@spreadshirt.net>

COPY ftp2s3 /usr/bin
ENV  FTP2S3_COMMANDS="--s3-region=eu-central-1 --s3-credentials=ACCESSKEY:SECRETKEY --s3-bucket='https://my-bucket.s3.amazonaws.com'"
VOLUME ["/etc/ftp2s3"]
EXPOSE 21

CMD ftp2s3 --ftp-addr=0.0.0.0:21 $FTP2S3_COMMANDS /etc/ftp2s3/credentials.txt
