#!/bin/bash -e

set -e
set -o pipefail

CONTAINER=catbase

OLD_ID=$(docker ps -f NAME="$CONTAINER" -q)

if [[ $OLD_ID -gt 0 ]]; then
	echo "Removing old container $OLD_ID"
	docker stop $CONTAINER
	docker rm -f $CONTAINER
fi
docker run \
  -d \
  -p 127.0.0.1:1337:1337 \
  -v var:/app/var \
  -v src:/app/src \
  -e TWITCHAUTHORIZATION="OAuth " \
  -e TWITCHCLIENTID="" \
  -e UNTAPPDTOKEN="" \
  -e HTTPADDR="0.0.0.0:1337" \
  -e TWITTERACCESSTOKEN="" \
  -e TWITTERACCESSSECRET="" \
  -e TWITTERCONSUMERKEY="" \
  -e TWITTERCONSUMERSECRET="" \
  -e AOCSESSION="" \
  -e TWILIOTOKEN="" \
  -e TWILIOSID="" \
  -e TWILIONUMBER="+1" \
  -e TYPE=slackapp \
  -e SLACKTOKEN= \
  -e SLACKUSERTOKEN= \
  -e SLACKVERIFICATION= \
  -e SLACKBOTID= \
  -e SLACKAPPLOGDIR=/app/var/logs \
  -e SLACKAPPLOGMESSAGEDIR=/app/var/logs \
  -e GOMAXPROCS=8 \
  --name $CONTAINER chrissexton/private:catbase

echo 'Subject: catbase built' | ssmtp noreply@velour.ninja
