#!/usr/bin/env bash
# 编译生成目标文件并且把文件同步到测试服务器上

cd $(dirname ${0}) # shell文件路径
cd ..              # 工程根目录

./build.sh -c 'linux:amd64'
mv Tomahawk_linux_amd64 Tomahawk

declare -a servers=('172.21.3.39' '172.21.3.92' '172.21.3.177')

for server in "${servers[@]}"
do
    echo "Sync ${server}"

    rsync -avz -e 'ssh -p 22' ./Tomahawk root@${server}:/home/tomahawk
    ssh root@${server} 'chown -R tomahawk:tomahawk /home/tomahawk/Tomahawk'

    rsync -avz -e 'ssh -p 22' ./tools/tomahawk.sh root@${server}:/home/tomahawk
    ssh root@${server} 'chown -R tomahawk:tomahawk /home/tomahawk/tomahawk.sh'

    rsync -avz -e 'ssh -p 22' ./tools/tomahawk.conf root@${server}:/home/tomahawk
    ssh root@${server} 'chown -R tomahawk:tomahawk /home/tomahawk/tomahawk.conf'
done

# 如果不想每一次都输入密码，只需要把本地的 ~/.ssh/id_rsa.pub 添加到服务器的 ~/.ssh/authorized_keys 中去就可以了
