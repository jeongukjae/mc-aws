#!/bin/bash
# amazon linux 2

set -ex

WEBHOOK_URL=
S3_PATH=
HOSTED_ZONE=
RECORD_NAME=

COMMENT="Auto updating @ `date`"
TTL=60

PUBLIC_IP=$(curl http://checkip.amazonaws.com)
cat > /file-to-change-ip << EOF
{
    "Comment":"$COMMENT",
    "Changes":[
        {
            "Action":"UPSERT",
            "ResourceRecordSet":{
                "ResourceRecords":[
                    {
                        "Value":"$PUBLIC_IP"
                    }
                ],
                "Name":"${RECORD_NAME}",
                "Type":"A",
                "TTL":$TTL
            }
        }
    ]
}
EOF

aws route53 change-resource-record-sets \
    --hosted-zone-id $HOSTED_ZONE \
    --change-batch file://file-to-change-ip
curl -X POST \
    -H 'Content-type: application/json' \
    --data "{\"content\":\"Updated ip of $RECORD_NAME to $PUBLIC_IP\"}" \
    "${WEBHOOK_URL}"

# https://docs.aws.amazon.com/AmazonECS/latest/developerguide/docker-basics.html
yum update -y
amazon-linux-extras install -y docker
yum install -y docker
service docker start
systemctl enable docker
usermod -a -G docker ec2-user

# run manager
mkdir /mc-server-data
docker run -d \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /mc-server-data:/mc-server-data \
    -p 80:80 \
    ghcr.io/jeongukjae/mc-aws-manager \
    -data /mc-server-data \
    -s3_path "${S3_PATH}" \
    -webhook "${WEBHOOK_URL}"
