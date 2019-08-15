#!/usr/bin/env bash
# 编译生成目标文件并且文件同步到测试服务器

cd $(dirname ${0}) # shell文件路径
cd ..              # 工程根目录

./build.sh -c 'linux:amd64'

declare -a servers=('172.21.3.39' '172.21.3.92' '172.21.3.177')

for server in "${servers[@]}"
do
    echo "Sync ${server}"
    rsync -avz -e 'ssh -p 22' ./Tomahawk_linux_amd64 root@${server}:/home/tomahawk
    ssh root@${server} 'chown -R tomahawk:tomahawk /home/tomahawk/Tomahawk_linux_amd64'
done
