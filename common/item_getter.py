from abc import ABC
from abc import abstractmethod
import sys
import threading

import grpc

sys.path.extend([".."])

from protos.proto_pb2 import Empty
from protos.proto_pb2_grpc import AnalysisStub


class ItemGetter(ABC):
    def __init__(self, ip: str, port: int) -> None:
        self.__ip = ip
        self.__port = port
        self.__thread = threading.Thread(target=self.__get_item, daemon=True)

    def __get_item(self) -> None:
        with grpc.insecure_channel(f"{self.__ip}:{self.__port}") as channel:
            stub = AnalysisStub(channel)
            self.get_item(stub)

    @abstractmethod
    def get_item(self, stub): ...

    def start(self) -> None:
        self.__thread.start()


class FrameGetter(ItemGetter):
    def get_item(self, stub):
        for frame in stub.get_image(Empty()):
            self.__frame = frame.data

    @property
    def frame(self) -> bytes:
        return self.__frame


class BoxGetter(ItemGetter):
    def get_item(self, stub):
        for box in stub.analysis(Empty()):
            self.__item = box.item

    @property
    def box(self) -> list[dict[str, int]]:
        return self.__item
