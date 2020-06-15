FROM alpine:edge

RUN apk add --no-cache git
RUN apk add --no-cache musl-dev
RUN apk add --no-cache gcc
RUN apk add --no-cache sqlite
RUN apk add --no-cache go
RUN apk add --no-cache perl
RUN apk add --no-cache make

VOLUME /app/var
VOLUME /app/src
EXPOSE 1337

ARG gomaxprocs="8"

WORKDIR /app

ENV SRC_DIR=/app/src/catbase/
RUN mkdir -p $SRC_DIR

ENV TWITCHAUTHORIZATION="OAuth "
ENV TWITCHCLIENTID=""
ENV UNTAPPDTOKEN=""
ENV HTTPADDR="0.0.0.0:1337"

ENV TWITTERACCESSTOKEN=""
ENV TWITTERACCESSSECRET=""
ENV TWITTERCONSUMERKEY=""
ENV TWITTERCONSUMERSECRET=""

ENV AOCSESSION=""

ENV TWILIOTOKEN=""
ENV TWILIOSID=""
ENV TWILIONUMBER="+5558675309"

ENV TYPE=slackapp
ENV SLACKTOKEN=FOO
ENV SLACKUSERTOKEN=FOO
ENV SLACKVERIFICATION=FOO
ENV SLACKBOTID=FOO

ENV SLACKAPPLOGDIR=/app/var/logs
ENV SLACKAPPLOGMESSAGEDIR=/app/var/logs

ENV GOMAXPROCS=8

ADD . $SRC_DIR

RUN apk add --no-cache tzdata
ENV TZ America/New_York

RUN git clone https://github.com/chrissexton/rank-amateur-cowsay.git cowsay && cd cowsay && ./install.sh
RUN cd $SRC_DIR; go get ./...; go build -o /app/catbase

RUN git clone https://gitlab.com/DavidGriffith/frotz.git frotz && cd frotz && make dfrotz && cp dfrotz /app

ENTRYPOINT ["/app/catbase", "-db=/app/var/catbase.db", "-debug"]
