#!/usr/bin/env bash

case $1 in
'1')
    ./Tomahawk --port 6301 --http 6201 --id node_1
    ;;
'2')
    ./Tomahawk --port 6302 --http 6202 --id node_2 --hosts 127.0.0.1:6301
    ;;
'3')
    ./Tomahawk --port 6303 --http 6203 --id node_3 --hosts 127.0.0.1:6301
    ;;
'4')
    ./Tomahawk --port 6304 --http 6204 --id node_4 --hosts 127.0.0.1:6301
    ;;
'5')
    ./Tomahawk --port 6305 --http 6205 --id node_5 --hosts 127.0.0.1:6301
    ;;
*)
    echo 'Unknown command'
    ;;
esac
