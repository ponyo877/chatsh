# ✅ List
grpcurl -plaintext localhost:50051 list

# ✅ CreateRoom
grpcurl -plaintext -import-path . -import-path grpc -proto grpc/chatsh.proto \
  -d '{"path": "/etc/test", "owner_token": "your_auth_token"}' \
  localhost:50051 fs.ChatshService/CreateRoom

# ✅ CreateDirectory
grpcurl -plaintext -import-path . -import-path grpc -proto grpc/chatsh.proto \
  -d '{"path": "/etc", "owner_token": "your_auth_token"}' \
  localhost:50051 fs.ChatshService/CreateDirectory

# ✅ DeletePath
grpcurl -plaintext -import-path . -import-path grpc -proto grpc/chatsh.proto \
  -d '{"path": "/etc", "owner_token": "your_auth_token"}' \
  localhost:50051 fs.ChatshService/DeletePath

# ✅ CopyPath
grpcurl -plaintext -import-path . -import-path grpc -proto grpc/chatsh.proto \
  -d '{"source_path": "/etc/test", "destination_path": "/var/test_bk", "owner_token": "your_auth_token"}' \
  localhost:50051 fs.ChatshService/CopyPath

grpcurl -plaintext -import-path . -import-path grpc -proto grpc/chatsh.proto \
  -d '{"source_path": "/etc/test", "destination_path": "/var/", "owner_token": "your_auth_token"}' \
  localhost:50051 fs.ChatshService/CopyPath

# ✅ MovePath
grpcurl -plaintext -import-path . -import-path grpc -proto grpc/chatsh.proto \
  -d '{"source_path": "/var/test_bk2", "destination_path": "/etc/", "owner_token": "your_auth_token"}' \
  localhost:50051 fs.ChatshService/MovePath

# ✅ ListNodes
grpcurl -plaintext -import-path . -import-path grpc -proto grpc/chatsh.proto \
  -d '{"path": "/"}' \
  localhost:50051 fs.ChatshService/ListNodes

# ✅ ListMessages
grpcurl -plaintext -import-path . -import-path grpc -proto grpc/chatsh.proto \
  -d '{"path": "/etc/test"}' \
  localhost:50051 fs.ChatshService/ListMessages

# StreamMessage (ストリーミングRPCのため、実行すると接続が維持され、サーバーからのメッセージが表示され続けます)
grpcurl -plaintext -import-path . -import-path grpc -proto grpc/chatsh.proto \
  -d '{"path": "/example/stream_chat_room", "initi_token": "your_auth_token", "follow": true}' \
  localhost:50051 fs.ChatshService/StreamMessage

# ✅ SearchMessage
grpcurl -plaintext -import-path . -import-path grpc -proto grpc/chatsh.proto \
  -d '{"path": "/etc/test", "pattern": "is"}' \
  localhost:50051 fs.ChatshService/SearchMessage

# ✅ WriteMessage
grpcurl -plaintext -import-path . -import-path grpc -proto grpc/chatsh.proto \
  -d '{"text_content": "Hello, this is a test message.", "destination_path": "/etc/test", "owner_token": "your_auth_token"}' \
  localhost:50051 fs.ChatshService/WriteMessage
