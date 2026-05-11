#!/usr/bin/env bash
set -euo pipefail
: "${S3_ENDPOINT:?need S3_ENDPOINT}"
: "${S3_BUCKET:?need S3_BUCKET}"
: "${S3_KEY_PREFIX:=loadtest/}"
aws --endpoint-url "$S3_ENDPOINT" s3 rm "s3://${S3_BUCKET}/${S3_KEY_PREFIX}" --recursive
