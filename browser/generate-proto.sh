#!/bin/bash

# Ensure output directory exists
mkdir -p src/generated

PROTOC_GEN_ES_PATH="./node_modules/.bin/protoc-gen-es"
PROTOC_GEN_CONNECT_ES_PATH="./node_modules/.bin/protoc-gen-connect-es"

echo "Generating TypeScript ES Modules for messages and Connect-ES client..."

protoc \
  --plugin="protoc-gen-es=${PROTOC_GEN_ES_PATH}" \
  --plugin="protoc-gen-connect-es=${PROTOC_GEN_CONNECT_ES_PATH}" \
  --es_out="src/generated" \
  --es_opt="target=ts" \
  --connect-es_out="src/generated" \
  --connect-es_opt="target=ts" \
  --proto_path=src/grpc \
  src/grpc/chatsh.proto

if [ $? -ne 0 ]; then
  echo "Error generating proto files with @bufbuild plugins."
  # Check if plugin executables exist
  if [ ! -f "${PROTOC_GEN_ES_PATH}" ]; then
    echo "Plugin executable not found at ${PROTOC_GEN_ES_PATH}."
  fi
  if [ ! -f "${PROTOC_GEN_CONNECT_ES_PATH}" ]; then
    echo "Plugin executable not found at ${PROTOC_GEN_CONNECT_ES_PATH}."
  fi
  exit 1
fi

echo "Proto files generated successfully."
echo "Generated files in src/generated:"
ls -l src/generated
echo ""

echo "Script finished. Generated files should be pure ES Modules using @bufbuild."
