import logging
from logging import Logger
from typing import TextIO

from colorlog import ColoredFormatter
from colorlog import StreamHandler


def logger_handler() -> StreamHandler[TextIO]:
    formatter = ColoredFormatter(
        "{green}{asctime}{reset} :: {bold_purple}{name:^13}{reset} :: {log_color}{levelname:^8}{reset} :: {bold_white}{message}",
        datefmt="%H:%M:%S",
        reset=True,
        log_colors={
            "INFO": "bold_cyan",
            "DEBUG": "bold_yellow",
            "WARNING": "bold_red,fg_thin_yellow",
            "ERROR": "bold_red",
            "CRITICAL": "bold_red,bg_white",
        },
        style="{",
    )
    handler = logging.StreamHandler()
    handler.setFormatter(formatter)
    return handler


def get_logger(name: str) -> Logger:
    logger = logging.getLogger(name)
    logger.setLevel(logging.INFO)
    logger.addHandler(logger_handler())
    return logger
