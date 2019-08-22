#!/usr/bin/env bash
# 更新tag并push到远程分支上
# 此处只会增加tag的最后一位，如果要修改大版本则需要手动操作

cd $(dirname ${0}) # shell文件路径
cd ..              # 工程根目录

git fetch --tags
latestTag=$(git describe --tags `git rev-list --tags --max-count=1`)
echo "Old tag: ${latestTag}"

# 拆分版本信息
IFS='.' read -ra tag <<< "${latestTag}"
# 将最后一位的版本号加一
tag[2]="$((tag[2] + 1))"
# 得到新的版本号
newTag="${tag[0]}.${tag[1]}.${tag[2]}"
echo "New tag: ${newTag}"

git tag ${newTag}
git push origin ${newTag}
