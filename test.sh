#!/bin/sh
set -e

msg="0123456789"
id=$(curl -X POST http://127.0.0.1:8080/api/new --data "$msg")
msgR=$(curl http://127.0.0.1:8080/api/$id)
if [ "$msgR" != "$msg" ]; then
  echo "非预期值"
  exit 1
fi
for i in $(seq 0 9); do
  echo;
  echo $i;
  sleep 2;
  msg="${msg%?}";
  curl -X POST http://127.0.0.1:8080/api/$id --data "$msg";
  msgR=$(curl http://127.0.0.1:8080/api/$id);
  if [ "$msgR" != "$msg" ]; then
    echo "非预期值";
    exit 1;
  fi
  echo $msgR;
  echo;
  # stat /home/saryta/tmp/shared-clipboard/clipboard-files/$id;
done
