#!/bin/sh

CHECKSUM=`shasum $2 -a 256 | cut -d ' ' -f 1`
SIGNATURE=`/usr/local/Cellar/openssl/1.0.2m/bin/openssl dgst -sha256 -sign $1 -hex $2 | cut -d ' ' -f 2`

echo "{\"checksum\": \"$CHECKSUM\", \"signature\": \"$SIGNATURE\"}"
