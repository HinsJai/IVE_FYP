from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Class(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = []
    HARDHAT: _ClassVar[Class]
    MASK: _ClassVar[Class]
    NO_HARDHAT: _ClassVar[Class]
    NO_MASK: _ClassVar[Class]
    NO_SAFETY_VEST: _ClassVar[Class]
    PERSON: _ClassVar[Class]
    SAFETY_CONE: _ClassVar[Class]
    SAFETY_VEST: _ClassVar[Class]
    MACHINERY: _ClassVar[Class]
    VEHICLE: _ClassVar[Class]
    BLUE_HARDHAT: _ClassVar[Class]
    ORANGE_HARDHAT: _ClassVar[Class]
    WHITE_HARDHAT: _ClassVar[Class]
    YELLOW_HARDHAT: _ClassVar[Class]
HARDHAT: Class
MASK: Class
NO_HARDHAT: Class
NO_MASK: Class
NO_SAFETY_VEST: Class
PERSON: Class
SAFETY_CONE: Class
SAFETY_VEST: Class
MACHINERY: Class
VEHICLE: Class
BLUE_HARDHAT: Class
ORANGE_HARDHAT: Class
WHITE_HARDHAT: Class
YELLOW_HARDHAT: Class

class Empty(_message.Message):
    __slots__ = []
    def __init__(self) -> None: ...

class Image(_message.Message):
    __slots__ = ["data"]
    DATA_FIELD_NUMBER: _ClassVar[int]
    data: bytes
    def __init__(self, data: _Optional[bytes] = ...) -> None: ...

class Item(_message.Message):
    __slots__ = ["x1", "y1", "x2", "y2", "class_type"]
    X1_FIELD_NUMBER: _ClassVar[int]
    Y1_FIELD_NUMBER: _ClassVar[int]
    X2_FIELD_NUMBER: _ClassVar[int]
    Y2_FIELD_NUMBER: _ClassVar[int]
    CLASS_TYPE_FIELD_NUMBER: _ClassVar[int]
    x1: int
    y1: int
    x2: int
    y2: int
    class_type: Class
    def __init__(self, x1: _Optional[int] = ..., y1: _Optional[int] = ..., x2: _Optional[int] = ..., y2: _Optional[int] = ..., class_type: _Optional[_Union[Class, str]] = ...) -> None: ...

class Response(_message.Message):
    __slots__ = ["item"]
    ITEM_FIELD_NUMBER: _ClassVar[int]
    item: _containers.RepeatedCompositeFieldContainer[Item]
    def __init__(self, item: _Optional[_Iterable[_Union[Item, _Mapping]]] = ...) -> None: ...

class LogRequest(_message.Message):
    __slots__ = ["message", "image"]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    IMAGE_FIELD_NUMBER: _ClassVar[int]
    message: str
    image: Image
    def __init__(self, message: _Optional[str] = ..., image: _Optional[_Union[Image, _Mapping]] = ...) -> None: ...

class NotificationRequest(_message.Message):
    __slots__ = ["camID", "class_type", "workplace"]
    CAMID_FIELD_NUMBER: _ClassVar[int]
    CLASS_TYPE_FIELD_NUMBER: _ClassVar[int]
    WORKPLACE_FIELD_NUMBER: _ClassVar[int]
    camID: str
    class_type: _containers.RepeatedScalarFieldContainer[Class]
    workplace: str
    def __init__(self, camID: _Optional[str] = ..., class_type: _Optional[_Iterable[_Union[Class, str]]] = ..., workplace: _Optional[str] = ...) -> None: ...
