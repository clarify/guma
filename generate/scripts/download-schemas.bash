#!/bin/bash
set -e

cd "$(dirname "$0")/../.."

DOWNLOAD_URL="https://opcfoundation.org/UA/schemas/1.03/"
TARGET_DIR="schemas/1.03"
mkdir -p "$TARGET_DIR"
cd "$TARGET_DIR"

for file in $(curl $DOWNLOAD_URL | grep href | sed 's/.*href="//' | sed 's/".*//' | grep '^[a-zA-Z].*')
do
	if [ -e "$file" ]
	then
		echo "File already downloaded: $file" 1>&2
		continue
	fi
	curl -s -O $DOWNLOAD_URL/$file
done
