#!/bin/env bash
# CI script for UBI8 job
# purpose: run unit test suite and generate code coverage report

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

# without glibc-langpack-en locale setting in CentOS8 is broken without this package
yum install -y git golang gcc make glibc-langpack-en qpid-proton-c-devel

export APPUTILSPATH=$PWD/apputils
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

go mod tidy

if [[ $FOUND_APPUTILS_BRANCH == 'success' ]]; then
  git clone https://github.com/infrawatch/apputils $APPUTILSPATH
  pushd $APPUTILSPATH
  git checkout "${GITHUB_REF#refs/heads/}"
  popd
  echo "replace $(grep apputils go.mod | tr '\n' ' ' | tr '\t' ' ')=> $APPUTILSPATH" >> go.mod
fi

go test -v -coverprofile=profile.cov ./...
