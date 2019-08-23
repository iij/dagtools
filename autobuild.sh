#!/bin/bash

make clean
make
cp /d/dagtools-windows-amd64-1.6.0/dagtools.ini ./build/dagtools--amd64-1.6.0/
rm ./build/dagtools--amd64-1.6.0/dagtools.ini.sample
mv ./build/dagtools--amd64-1.6.0/dagtools ./build/dagtools--amd64-1.6.0/dagtools.exe
