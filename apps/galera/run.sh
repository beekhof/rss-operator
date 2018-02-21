#!/bin/bash


_term() { 
  echo "Caught SIGTERM signal!" 
  exit 0
}

trap _term SIGTERM

echo "$$ Container running"
while [ 1 = 1 ]; do
    printf .
    sleep 30
done
echo "$$ Container done."
