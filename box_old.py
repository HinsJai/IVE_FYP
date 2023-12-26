# import base64
# import contextlib
# import logging
# import threading
# import time
# from concurrent import futures
# from math import ceil
# from typing import Self

# import cv2
# import grpc
# import numpy as np
# from typing_extensions import override
# from ultralytics import YOLO

# import proto_pb2
# import proto_pb2_grpc
# from frame_getter import FrameGetter
# from logger import get_logger


# class AnalysisServer:

#     def __init__(
#         self, ip: str, port: int, cam_server_ip: str, cam_server_port: int
#     ) -> None:
#         self.__ip = ip
#         self.__port = port
#         self.__logger = get_logger("Model Server")
#         self.__server = grpc.server(futures.ThreadPoolExecutor())
#         self.__frame_getter = FrameGetter(cam_server_ip, cam_server_port)
#         # self.__model = YOLO("ppe.pt")


#     def __enter__(self) -> Self:
#         proto_pb2_grpc.add_AnalysisServicer_to_server(
#             AnalysisService(self), self.__server
#         )
#         self.__server.add_insecure_port(f"{self.__ip}:{self.__port}")
#         return self

#     def log(self, info: str) -> None:
#         self.__logger.info(info)

#     def __exit__(self, _, __, ___) -> None:
#         self.__server.stop(None)

#     def start(self) -> None:
#         self.__logger.info(f"Server started at {self.__ip}:{self.__port}")
#         logging.info(f"Server started at {self.__ip}:{self.__port}")
#         self.__server.start()
#         self.__frame_getter.start()
#         self.__thread = threading.Thread(target=self.__get_box, daemon=True)
#         self.__thread.start()
#         # print(self.__frame_getter.frame)
#         with contextlib.suppress(KeyboardInterrupt):
#             self.__server.wait_for_termination()

#     def __get_box(self) -> None:
#         with grpc.insecure_channel(f"{self.__ip}:{self.__port}") as channel:
#             stub = proto_pb2_grpc.AnalysisStub(channel)
#             for box in stub.analysis(proto_pb2.Empty()):
#                 # stub.analysis(proto_pb2.Empty())

#                 self.box = box.item
#             # print(f"self.box {self.box}")

#     @property
#     def boxes(self):
#         # print("boxes called")
#         self.frame_data = base64.b64decode(self.__frame_getter.frame)
#         self.np_data = np.frombuffer(self.frame_data, dtype=np.uint8)
#         self.frame = cv2.imdecode(self.np_data, flags=1)
#         result = self.__model.predict(self.frame)

#         if result and len(result) > 0:
#             # Access the 'boxes' attribute of the first item in the list
#             boxes = result[0].boxes
#             # print(result)
#             return [
#                 {
#                     "x1": int(box.xyxy[0][0]),
#                     "y1": int(box.xyxy[0][1]),
#                     "x2": int(box.xyxy[0][2]),
#                     "y2": int(box.xyxy[0][3]),
#                     "class_type": int(box.cls[0]),
#                 }
#                 for box in boxes if ceil((box.conf[0] * 100)) / 100 > 0.5
#             ]
#         else:
#             return []  # Return an empty list if no boxes are detected


# class AnalysisService(proto_pb2_grpc.AnalysisServicer):
#     def __init__(self, server: AnalysisServer) -> None:
#         super().__init__()
#         self.__server = server

#     # @override
#     # def analysis(self, request, context):
#     #     self.__server.log(f"{context.peer()} connected")
#     #     while True:
#     #         # boxes = self.__server.boxes  # 调用 boxes 属性获取结果
#     #         # response = proto_pb2.Response(item=boxes)  # 创建 Response 对象
#     #         # print(response)
#     #         # yield response
#     #         yield proto_pb2.Response(self.__server.boxes)
#     #         time.sleep(1 / 60)
#     @override
#     def analysis(self, request, context):
#         self.__server.log(f"{context.peer()} connected")
#         while True:
#             boxes = self.__server.boxes  # Call the 'boxes' method to get the detected boxes
#             # Create a Response object with the 'data' attribute
#             response = proto_pb2.Response(item=boxes)
#             yield response
#             time.sleep(1 / 60)


# if __name__ == "__main__":
#     ip = "localhost"
#     port = 8787
#     cam_server_ip = "localhost"
#     cam_server_port = 8877
#     with AnalysisServer(ip, port, cam_server_ip, cam_server_port) as server:
#         server.start()
