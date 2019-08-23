[![logo](logo.png)](https://github.com/RitterHou/Tomahawk)

# Tomahawk

[![Build Status](https://travis-ci.org/RitterHou/Tomahawk.svg?branch=master)](https://travis-ci.org/RitterHou/Tomahawk)
![](https://img.shields.io/github/tag/RitterHou/Tomahawk.svg)
![Golang](https://img.shields.io/badge/golang-1.12.5-blue.svg)
[![GitHub license](https://img.shields.io/github/license/RitterHou/Tomahawk)](https://github.com/RitterHou/Tomahawk/blob/master/LICENSE)

### Installation

Download the [releases](https://github.com/RitterHou/Tomahawk/releases/latest) for your platform or build the source code by yourself.

### Launch

Startup process with [tomahawk.conf](https://github.com/RitterHou/Tomahawk/blob/master/tools/tomahawk.conf)

    ./Tomahawk -c tomahawk.conf

You can also pass params in command line if you don't like using configuration files

    ./Tomahawk --level info --port 6301 --http 6201 --id node_1 --quorum 2 \
    --hosts 127.0.0.1:6301,127.0.0.1:6302,127.0.0.1:6303

However, you can use command line params and configuration file params at the same time, just remember the params from command line have higher priority than configuration file, this means the params you given in command line will override the values from the configuration file.

    Command Line > Configuration File > Default Values

Show help messages with

    ./Tomahawk -h

[tomahawk.sh](https://github.com/RitterHou/Tomahawk/blob/master/tools/tomahawk.sh) is a convenient way to start and stop Tomahawk process

    ./tomahawk.sh -f Tomahawk_linux_amd64 -p "-c tomahawk.conf --id node_1 --level info"

### Usage

Save data

    curl --header "Content-Type: application/json" \
         --request POST \
         --data '{"key": "city", "value": "Nanjing"}' \
         http://localhost:6200/entries
         
or data list

    curl --header "Content-Type: application/json" \
         --request POST \
         --data '[{"key": "city", "value": "Nanjing"}, {"key": "province", "value": "Jiangsu"}]' \
         http://localhost:6200/entries
         
Query data

    curl -X GET 'http://localhost:6202/entries?key=city'
    
Show nodes info

    curl -X GET 'http://localhost:6202/nodes'
    
### TODO

- [ ] Data persistence
- [ ] Data compression and snapshot creation
- [ ] More tests for special situations
