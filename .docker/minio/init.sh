#!/bin/sh
set -e

/usr/bin/minio server /data --console-address :9001 &
MINIO_PID=$!
until nc -z localhost 9000; do
  sleep 0.5
done

: "${MINIO_NOTIFY_WEBHOOK_ENABLE_1:=true}"
: "${MINIO_NOTIFY_WEBHOOK_ENDPOINT_1:=http://app:8080/files/webhook-minio}"
: "${MINIO_NOTIFY_WEBHOOK_AUTH_TOKEN_1:=dev-webhook-secret}"

mc config host add myminio http://localhost:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" --api S3v4
mc mb -p myminio/"${S3_BUCKET:-keeper}" || true
mc event add myminio/"${S3_BUCKET:-keeper}" arn:minio:sqs::1:webhook --event put,post,copy,multipart || true

cat > /tmp/policy.json <<'POL'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {"AWS": "*"},
      "Action": ["s3:PutObject","s3:AbortMultipartUpload","s3:ListBucketMultipartUploads","s3:ListBucket"],
      "Resource": [
        "arn:aws:s3:::${S3_BUCKET:-keeper}",
        "arn:aws:s3:::${S3_BUCKET:-keeper}/*"
      ],
      "Condition": {
        "IpAddress": {"aws:SourceIp": ["172.18.0.0/16","127.0.0.1/32"]}
      }
    }
  ]
}
POL
mc anonymous set none myminio/"${S3_BUCKET:-keeper}" || true
mc policy set-json /tmp/policy.json myminio/"${S3_BUCKET:-keeper}" || true

wait $MINIO_PID
