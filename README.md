[![logo](logo.png)](https://github.com/RitterHou/Tomahawk)

# Tomahawk

[![Build Status](https://travis-ci.org/RitterHou/Tomahawk.svg?branch=master)](https://travis-ci.org/RitterHou/Tomahawk)
![](https://img.shields.io/github/tag/RitterHou/Tomahawk.svg)
![Golang](https://img.shields.io/badge/golang-1.12.5-blue.svg)
[![GitHub license](https://img.shields.io/github/license/RitterHou/Tomahawk)](https://github.com/RitterHou/Tomahawk/blob/master/LICENSE)

### Installation

Download the [releases](https://github.com/RitterHou/Tomahawk/releases/latest) for your platform or build the source code by yourself.

### Usage

Startup process

    ./Tomahawk --level info --port 6301 --http 6201 --id node_1 --quorum 2 \
    --hosts 127.0.0.1:6301 --hosts 127.0.0.1:6302 --hosts 127.0.0.1:6303

Parameters

| parameter | purpose |
| --- | --- |
| level | log level, error/warn/info/debug, default is debug |
| port | the port for TCP listening, default is 6300 |
| http | the port for HTTP listening, default is 6200 |
| id | node unique id, default is random string with length 10 |
| quorum | quorum means most, using for election, default is 1 |
| hosts | seed hosts with other nodes, default is \[\] |

Save data on the leader node

    curl --header "Content-Type: application/json" \
         --request POST \
         --data '{"key": "city", "value": "Nanjing"}' \
         http://localhost:6200/entries
         
or saving datas

    curl --header "Content-Type: application/json" \
         --request POST \
         --data '[{"key": "city", "value": "Nanjing"}, {"key": "province", "value": "Jiangsu"}]' \
         http://localhost:6200/entries
         
Query data on every node by key

    curl -X GET 'http://localhost:6202/entries?key=city'
    
Show nodes info

    curl -X GET 'http://localhost:6202/nodes'
