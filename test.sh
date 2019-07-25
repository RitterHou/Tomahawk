#!/usr/bin/env bash

case $1 in
'1')
    ./Tomahawk --port 6301
    ;;
'2')
    ./Tomahawk --hosts 127.0.0.1:6301 --port 6302
    ;;
'3')
    ./Tomahawk --hosts 127.0.0.1:6301 --port 6303
    ;;
'4')
    ./Tomahawk --hosts 127.0.0.1:6301 --port 6304
    ;;
*)
    echo 'Unknown command'
    ;;
esac
