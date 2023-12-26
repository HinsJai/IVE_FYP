import sys
import tomllib
from typing import Any, Self
import io
import discord
from discord import app_commands

sys.path.extend([".."])

import asyncio
from dataclasses import dataclass, field

from protos.proto_pb2 import Empty
from protos.proto_pb2_grpc import DiscordLogServicer, add_DiscordLogServicer_to_server

try:
    from typing import Self, override
except ImportError:
    from typing_extensions import Self, override

from concurrent import futures

import grpc

# from discord.ext import commands

with open("config.toml", "rb") as config:
    config = tomllib.load(config)

GUILD_ID = config["discord"]["guild_id"]
LOG_CHANNEL = config["discord"]["log_channel"]
TOKEN = config["discord"]["token"]
DISCORD_LOGGER_URL = config["discord"]["ip"]
DISCORD_LOGGER_PORT = config["discord"]["port"]
INTERACTION_USER_ID = config["discord"]["interaction_user_id"]

del config

# MY_GUILD = discord.Object(id=GUILD_ID)


class MyClient(discord.Client):
    def __init__(self, *, intents: discord.Intents) -> None:
        super().__init__(intents=intents)
        self.tree = app_commands.CommandTree(self)
        self.__my_guild = discord.Object(id=GUILD_ID)

    async def on_message(self, message):
        # don't respond to ourselves
        if message.author == self.user:
            return

        if message.content == "ping":
            await message.channel.send("pong")

    async def setup_hook(self):
        # This copies the global commands over to your guild.
        self.tree.copy_global_to(guild=self.__my_guild)
        await self.tree.sync(guild=self.__my_guild)


@dataclass(slots=True, repr=False)
class DiscordBotServer:
    # __ip: str = field(init=False)
    # DISCORD_LOGGER_PORT: int = field(init=False)
    # __token: str = field(init=False)
    # guild_id: int = field(init=False)
    # log_channel_name: str
    __my_guild = discord.Object(id=GUILD_ID)
    __client: MyClient = field(init=False)
    __server: grpc.server = field(init=False)
    loop: any = field(init=False)

    def __post_init__(self) -> None:
        # self.__ip = DISCORD_LOGGER_URL
        # self.DISCORD_LOGGER_PORT = DISCORD_LOGGER_PORT
        # self.__token = TOKEN
        intents = discord.Intents.default()
        intents.message_content = True
        self.__client = MyClient(intents=intents)
        self.__client.event(self.on_ready)
        self.__client.tree.command(guild=self.__my_guild)(self.say)
        self.__server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))

    def start(self) -> None:
        self.__server.start()
        self.__client.run(TOKEN)

    def __enter__(self) -> Self:
        add_DiscordLogServicer_to_server(DiscordLogService(self), self.__server)
        self.__server.add_insecure_port(f"{DISCORD_LOGGER_URL}:{DISCORD_LOGGER_PORT}")
        return self

    def __exit__(self, _, __, ___) -> None:
        self.__server.stop(None)
        asyncio.run(self.__client.close())

    async def on_ready(self) -> None:
        self.loop = asyncio.get_running_loop()
        print(f"Logged in as {self.__client.user} (ID: {self.__client.user.id})")
        print("------")

    async def say(self, interaction: discord.Interaction, text_to_send: str) -> None:
        if interaction.user.id == INTERACTION_USER_ID:
            if interaction.channel is not None:
                await interaction.channel.send(text_to_send.replace("\\n", "\n"))
                await interaction.response.send_message(
                    "success", ephemeral=True, delete_after=5
                )
        else:
            await interaction.response.send_message(
                "You are not my owner", ephemeral=True, delete_after=5
            )

    async def send_log(self, log: str, image: bytes) -> None:
        guild = self.__client.get_guild(GUILD_ID)
        channel = discord.utils.get(guild.channels, name=LOG_CHANNEL)
        # await channel.send(log)
        data = io.BytesIO(image)

        await channel.send(log, file=discord.File(data, "frame.png"))


class DiscordLogService(DiscordLogServicer):
    def __init__(self, server: DiscordBotServer) -> None:
        super().__init__()
        self.__server = server

    @override
    def log(self, request, context) -> Empty:
        asyncio.run_coroutine_threadsafe(
            self.__server.send_log(request.message, request.image.data), self.__server.loop
        )
        return Empty()


if __name__ == "__main__":
    with DiscordBotServer() as server:
        server.start()
