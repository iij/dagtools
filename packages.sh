#!/bin/bash

set -eux

package=${1:-"github.com/iij/dagtools"}

rm -fr build
for os in "windows" "linux" "darwin"; do
    for arch in "386" "amd64"; do
        APP_PACKAGE=${package} OS_TYPE=${os} ARCH_TYPE=${arch} make
    done
done

