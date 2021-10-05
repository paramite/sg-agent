#!/bin/env bash
# CI script for UBI8 job
# purpose: spawn sg-core with scheduler plugin and verify task executions are being requested

set -ex

# enable required repo(s)
cat > /etc/yum.repos.d/fedora-eln.repo <<EOF
[centos-opstools]
name=opstools
baseurl=http://mirror.centos.org/centos/8/opstools/\$basearch/collectd-5/
gpgcheck=0
enabled=1
module_hotfixes=1
EOF

dnf install -y git golang gcc make qpid-proton-c-devel python3

export APPUTILSPATH=$PWD/apputils
export COREPATH=$PWD/sg-core

mkdir -p /usr/lib64/sg-core

# install sg-core
git clone https://github.com/infrawatch/sg-core $COREPATH
pushd $COREPATH

if [[ $FOUND_CORE_BRANCH == 'success' ]]; then
  git checkout "${GITHUB_REF#refs/heads/}"
fi

go mod tidy

if [[ $FOUND_APPUTILS_BRANCH == 'success' ]]; then
  git clone https://github.com/infrawatch/apputils $APPUTILSPATH
  pushd $APPUTILSPATH
  git checkout "${GITHUB_REF#refs/heads/}"
  popd
  echo "replace $(grep apputils go.mod | tr '\n' ' ' | tr '\t' ' ')=> $APPUTILSPATH" >> go.mod
fi

PRODUCTION_BUILD=true PLUGIN_DIR=/usr/lib64/sg-core/ ./build.sh
popd

# sync go.mod with local sg-core and apputils
sed -i "s/go [0-9\.]\{1,\}/$(grep -e '^go [0-9\.]' $COREPATH/go.mod)/" go.mod
echo "replace $(grep sg-core go.mod | tr '\n' ' ' | tr '\t' ' ')=> $COREPATH" >> go.mod
if [[ $FOUND_APPUTILS_BRANCH == 'success' ]]; then
  echo "replace $(grep apputils go.mod | tr '\n' ' ' | tr '\t' ' ')=> $APPUTILSPATH" >> go.mod
fi

# build sg-agent
PLUGIN_DIR=/usr/lib64/sg-core/ ./build.sh

# produce some events
/tmp/sg-core -config ./ci/smoke/config.yaml &>/tmp/sg-core.log &
CORE_PID=$!
sleep 12
kill -9 $CORE_PID

# debug output
cat /tmp/sg-core.log
cat /tmp/sg-agent.log

# validate produced events
./ci/smoke/validate.py /tmp/sg-agent.log
