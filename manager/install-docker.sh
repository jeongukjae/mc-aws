#!/bin/bash
# amazon linux 2
# https://docs.aws.amazon.com/AmazonECS/latest/developerguide/docker-basics.html

set -ex

yum update -y
amazon-linux-extras install -y docker
yum install -y docker
service docker start
systemctl enable docker
usermod -a -G docker ec2-user
