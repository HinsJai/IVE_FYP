py -3.11 -m grpc_tools.protoc -I. --python_out=./protos --pyi_out=./protos --grpc_python_out=./protos proto.proto &
protoc --go_out=./protos --go_opt=paths=source_relative --go-grpc_out=./protos --go-grpc_opt=paths=source_relative proto.proto

# after update protoc, change the proto_pb2_grpc.py to below
# import protos.proto_pb2 as proto__pb2
