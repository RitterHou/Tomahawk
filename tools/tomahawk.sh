#!/usr/bin/env bash
# 通过脚本的方式启动或停止Tomahawk进程

# 可执行文件的名称
file="Tomahawk"
# 程序执行时日志输出到指定文件
log="Tomahawk.log"
# 程序的工作文件夹
dir=$(dirname ${0})
# 可执行文件工作所需要的参数
params="-c tomahawk.conf"
# 操作类型
operation=""

pidFile="Tomahawk.pid"

# 默认打印帮助信息
if test $# -eq 0; then
    set -- "-h"
fi

# 循环获取参数信息
while test $# -gt 0; do
    case "$1" in
    -h|--help)
        echo "Tomahawk shell"
        echo " "
        echo "arguments:"
        echo "  start          start tomahawk daemon"
        echo "  stop           stop  tomahawk daemon"
        echo " "
        echo "options:"
        echo "  -h, --help     show help messages"
        echo "  -d, --dir      working director, default is shell script director"
        echo "  -f, --file     tomahawk executable file name, default is Tomahawk"
        echo "  -l, --log      specify a log file, default is Tomahawk.log"
        echo "  -p, --params   tomahawk running params, default is \"-c tomahawk.conf\""
        exit 0
        ;;
    -d|--dir)
        shift # 参数左移一个
        if test $# -gt 0; then
            dir=$1
        else
            echo "no director specify"
            exit 1
        fi
        shift
        ;;
    -f|--file)
        shift
        if test $# -gt 0; then
            file=$1
        else
            echo "no file specify"
            exit 1
        fi
        shift
        ;;
    -l|--log)
        shift
        if test $# -gt 0; then
            log=$1
        else
            echo "no log specify"
            exit 1
        fi
        shift
        ;;
    -p|--params)
        shift
        if test $# -gt 0; then
            params=$1
        else
            echo "no params specify"
            exit 1
        fi
        shift
        ;;
    start)
        operation="start"
        shift
        ;;
    stop)
        operation="stop"
        shift
        ;;
    *)
        break
        ;;
    esac
done

# 进入指定目录
cd ${dir}

if [[ "${operation}" = "start" ]]; then
    # 如果进程已经被启动，则不需要再次启动
    if [[ -e "${pidFile}" ]]; then
        echo -n "Tomahawk daemon is running, process id: "
        cat ${pidFile}
        echo ""
        exit 1
    fi

    ./${file} ${params} >${log} 2>&1 &
    pid=$!

    echo "Starting Tomahawk Daemon..."
    sleep 1

    # 检测进程是否启动成功
    ps -ef | grep ${pid} | grep -v grep
    if [[ $? -ne 0 ]]
    then
        echo "Start Tomahawk Failed"
        cat ${log}
    else
        echo "dir:    ${dir}"
        echo "file:   ${file}"
        echo "log:    ${log}"
        echo "params: ${params}"
        echo "Start Tomahawk Success"
        echo -n ${pid} > "${pidFile}"
    fi
elif [[ "${operation}" = "stop" ]]; then
    if [[ ! -e "${pidFile}" ]]; then
        echo "No pid file found"
        exit 1
    fi

    ps -ef | grep `cat ${pidFile}` | grep -v grep
    if [[ $? -ne 0 ]]
    then
        echo -n "Pid is not running: "
        cat ${pidFile}
        echo ""
        exit 1
    fi

    kill -9 `cat ${pidFile}`
    rm "${pidFile}"
    echo "Stop Tomahawk Success"
else
    echo "No operation can be checked: [start, stop]"
fi
