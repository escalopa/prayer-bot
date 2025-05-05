#!/bin/bash

# This script fetches the `access token`, `cloudID`, and `folderID` from Yandex Cloud
# and stores them in a `terraform.tfvars` file.

set -e

# check input arguments
if [ "$#" -ne 2 ]; then
  echo "usage: $0 <CLOUD_NAME> <FOLDER_NAME>"
  exit 1
fi

CLOUD_NAME="$1"
FOLDER_NAME="$2"
TFVARS_FILE="terraform.tfvars"


# get access token
echo "fetching access token"
ACCESS_TOKEN=$(yc iam create-token)
if [ -z "$ACCESS_TOKEN" ]; then
  echo "error: failed to get access token"
  exit 1
fi

# get cloud_id by cloud name
echo "fetching cloud id for cloud name '$CLOUD_NAME'"
CLOUD_ID=$(curl -s -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  https://resource-manager.api.cloud.yandex.net/resource-manager/v1/clouds \
  | jq -r --arg CLOUD_NAME "$CLOUD_NAME" '.clouds[] | select(.name == $CLOUD_NAME) | .id')

if [ -z "$CLOUD_ID" ]; then
  echo "error: could not find cloud with name '$CLOUD_NAME'"
  exit 1
fi

# get folder_id by folder name
echo "fetching folder id for folder name '$FOLDER_NAME' in cloud '$CLOUD_NAME'"
FOLDER_ID=$(curl -s -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -G https://resource-manager.api.cloud.yandex.net/resource-manager/v1/folders \
  --data-urlencode "cloudId=${CLOUD_ID}" \
  | jq -r --arg FOLDER_NAME "$FOLDER_NAME" '.folders[] | select(.name == $FOLDER_NAME) | .id')

if [ -z "$FOLDER_ID" ]; then
  echo "error: could not find folder with name '$FOLDER_NAME' in cloud '$CLOUD_NAME'"
  exit 1
fi

# replace values in terraform.tfvars
echo "updating $TFVARS_FILE"

echo "access_token = \"$ACCESS_TOKEN\"" > terraform.tfvars
echo "cloud_id = \"$CLOUD_ID\"" >> terraform.tfvars
echo "folder_id = \"$FOLDER_ID\"" >> terraform.tfvars

echo "terraform.tfvars updated successfully"

