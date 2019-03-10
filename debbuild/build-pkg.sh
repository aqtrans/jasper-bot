#!/bin/sh
cd ../
go build -o debbuild/jasper-bot
cd debbuild/
debuild -us -uc -b
