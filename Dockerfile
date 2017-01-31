FROM debian:latest
MAINTAINER Andreas Linz <anli@spreadshirt.net>

COPY ftp2s3 /usr/bin
ENV  FTP2S3_COMMANDS=""
VOLUME ["/etc/ftp2s3", "/var/ftproot"]
EXPOSE 21

CMD ftp2s3 --ftp-root=/var/ftproot --ftp-addr=0.0.0.0:21 $FTP2S3_COMMANDS /etc/ftp2s3/credentials.txt
