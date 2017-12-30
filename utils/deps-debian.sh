#!/bin/bash
apt-get install --no-install-recommends -y git-core ca-certificates
apt-get install --no-install-recommends -y puppet-common
# needed for some puppet facts
apt-get install --no-install-recommends -y lsb-release
