#!/bin/sh

# Save alias and poll connection
until mc alias set raggy http://s3:9000 admin miniopassword; do
  echo 'Waiting for S3 server...'
  sleep 1
done

echo 'Creating bucket: uploads'
mc mb --ignore-existing raggy/uploads

mc anonymous set private raggy/uploads

echo 'Adding event hook: uploads->put'
mc event add raggy/uploads arn:minio:sqs::primary:kafka --event put

echo 'S3 server initialization complete.'
exit 0
