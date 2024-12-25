# we use aiogram 3.x!
from aiogram import Bot, Dispatcher, types, F
from aiogram.filters import Command, CommandStart
import httpx
import os
import dotenv

dotenv.load_dotenv()
token = os.getenv('BOT_TOKEN')
api_url = os.getenv('API_URL')
bot = Bot(token=token)
dp = Dispatcher(bot)

@dp.message(CommandStart())
async def start(message: types.Message, command: Command):
    await message.answer('Привет! Я бот, который поможет вам вести список джазовых стандартов. Для начала проверим, есть ли у вас аккаунт по имени пользователя из telegram.')
    # get user info
    user = message.from_user
    username = user.username
    user_id = user.id  # this will be crucial if a user doesn't have a username, which is not rare
    with httpx.AsyncClient() as client:
        response = await client.post(f'{api_url}/users', json={'username': username, 'user_id': user_id})