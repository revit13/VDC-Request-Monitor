#!/bin/sh

exec sh /run.sh &
sleep 20
exec curl -k 'http://127.0.0.1:8000'

