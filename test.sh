#!/usr/bin/env bash

case $1 in
'1')
    ./Tomahawk --port 6301 --http 6201 --id 1
    ;;
'2')
    ./Tomahawk --port 6302 --http 6202 --id 2 --hosts 127.0.0.1:6301
    ;;
'3')
    ./Tomahawk --port 6303 --http 6203 --id 3 --hosts 127.0.0.1:6301
    ;;
'4')
    ./Tomahawk --port 6304 --http 6204 --id 4 --hosts 127.0.0.1:6301
    ;;
*)
    echo 'Unknown command'
    ;;
esac
