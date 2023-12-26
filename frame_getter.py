# import threading

# import grpc

# import proto_pb2
# import proto_pb2_grpc
# from item_getter import ItemGetter

# # class FrameGetter:
# #     def __init__(self, ip: str, port: int) -> None:
# #         self.__ip = ip
# #         self.__port = port
# #         self.__frame = None
# #         self.__thread = threading.Thread(target=self.__get_frame, daemon=True)

# #     def __get_frame(self) -> None:
# #         with grpc.insecure_channel(f"{self.__ip}:{self.__port}") as channel:
# #             stub = proto_pb2_grpc.AnalysisStub(channel)
# #             for frame in stub.get_image(proto_pb2.Empty()):
# #                 self.__frame = frame.data

# #     def start(self) -> None:
# #         self.__thread.start()

# #     @property
# #     def frame(self) -> bytes:
# #         return self.__frame

# class FrameGetter(ItemGetter):
#     def get_item(self, stub):
#         for frame in stub.get_image(proto_pb2.Empty()):
#                 self.__item = frame.data