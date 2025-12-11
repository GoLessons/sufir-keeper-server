#!/bin/sh
set -e

/usr/bin/docker-entrypoint.sh "$@" &
MINIO_PID=$!

mc alias set myminio http://localhost:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" --api S3v4 || true
until mc ready myminio >/dev/null 2>&1; do
  sleep 0.5
done
mc mb -p myminio/"${S3_BUCKET:-keeper}" || true
# Removed 'multipart' and 'post' event type as it can cause issues with some mc versions
mc event add myminio/"${S3_BUCKET:-keeper}" arn:minio:sqs::1:webhook --event put || true
mc anonymous set none myminio/"${S3_BUCKET:-keeper}" || true

wait $MINIO_PID
