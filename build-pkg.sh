#!/bin/sh
go build -o jasper-bot
debuild -us -uc -b
