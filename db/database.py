import os
from contextlib import contextmanager
from sqlalchemy import create_engine
from sqlalchemy.orm import DeclarativeBase, sessionmaker


DB_FILE = os.getenv("JAZZ_DB_FILE", "jazz.db")
if DB_FILE == ":memory:":  # use in-memory database
    SQLALCHEMY_DATABASE_URL = "sqlite:///:memory:"
else:
    SQLALCHEMY_DATABASE_URL = f"sqlite:///./{DB_FILE}"

# Создание движка SQLAlchemy с подключением к базе данных SQLite
engine = create_engine(
    SQLALCHEMY_DATABASE_URL,
    connect_args={"check_same_thread": False}
)

# Создание фабрики сессий с привязкой к созданному движку
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)

# Базовый класс для объявления моделей
class Base(DeclarativeBase):
    pass

def init_db():
    from db import models  # to load the base classes
    Base.metadata.create_all(bind=engine)

# Функция-генератор для получения сессии, автоматически закрывает сессию после использования
#@contextmanager
def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
