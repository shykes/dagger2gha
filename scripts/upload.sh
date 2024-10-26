#!/bin/bash

set -e

# Usage: ./upload_artifact.sh <file_path> [artifact_name]

FILE_PATH=$1
ARTIFACT_NAME=${2:-artifact}

if [ -z "$FILE_PATH" ]; then
  echo "Usage: $0 <file_path> [artifact_name]"
  exit 1
fi

if [ ! -f "$FILE_PATH" ]; then
  echo "File not found: $FILE_PATH"
  exit 1
fi

# Required environment variables provided by GitHub Actions
ACTIONS_RUNTIME_URL=${ACTIONS_RUNTIME_URL:?ACTIONS_RUNTIME_URL not set}
ACTIONS_RUNTIME_TOKEN=${ACTIONS_RUNTIME_TOKEN:?ACTIONS_RUNTIME_TOKEN not set}
RUN_ID=${GITHUB_RUN_ID:?GITHUB_RUN_ID not set}

echo "Uploading '$FILE_PATH' as artifact '$ARTIFACT_NAME'..."

# Create an artifact container
CREATE_ARTIFACT_URL="${ACTIONS_RUNTIME_URL}_apis/pipelines/workflows/${RUN_ID}/artifacts?api-version=6.0-preview"
CREATE_RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer $ACTIONS_RUNTIME_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"Type\": \"actions_storage\", \"Name\": \"${ARTIFACT_NAME}\"}" \
  "$CREATE_ARTIFACT_URL")

# Extract the upload URL
UPLOAD_URL=$(echo "$CREATE_RESPONSE" | jq -r '.fileContainerResourceUrl')
if [ -z "$UPLOAD_URL" ] || [ "$UPLOAD_URL" == "null" ]; then
  echo "Failed to create artifact container."
  exit 1
fi

# Upload the file
FILE_NAME=$(basename "$FILE_PATH")
UPLOAD_FILE_URL="${UPLOAD_URL}?itemPath=${ARTIFACT_NAME}%2F${FILE_NAME}"
curl -s -X PUT \
  -H "Authorization: Bearer $ACTIONS_RUNTIME_TOKEN" \
  -H "Content-Type: application/octet-stream" \
  --upload-file "$FILE_PATH" \
  "$UPLOAD_FILE_URL" >/dev/null

# Finalize the artifact
FINALIZE_RESPONSE=$(curl -s -X PATCH \
  -H "Authorization: Bearer $ACTIONS_RUNTIME_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"Type\": \"actions_storage\", \"Name\": \"${ARTIFACT_NAME}\"}" \
  "$CREATE_ARTIFACT_URL")

echo "Artifact '$ARTIFACT_NAME' uploaded successfully."
