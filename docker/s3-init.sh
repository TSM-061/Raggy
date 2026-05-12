#!/bin/sh

# Save alias and poll connection
until /usr/bin/mc alias set raggy http://s3:9000 admin miniopassword; do
  echo 'Waiting for S3 server...'
  sleep 1
done

# Create the bucket
echo 'Creating bucket: uploads'
/usr/bin/mc mb --ignore-existing raggy/uploads

# Optional: Set the bucket to public or private
/usr/bin/mc anonymous set private raggy/uploads

echo 'S3 server initialization complete.'
exit 0
