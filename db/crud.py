from datetime import datetime

from sqlalchemy.orm import Session
from sqlalchemy import select, update, delete
from sqlalchemy import func
from sqlalchemy.exc import SQLAlchemyError
import logging

from db import models

# Настройка логирования
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def create_user(db: Session, username: str, name: str):
    db_user = models.User(username=username, name=name)
    db.add(db_user)
    db.commit()
    db.refresh(db_user)
    return db_user

def get_user(db: Session, user_id: int = None, username: str = None):
    query = select(models.User)
    if user_id:
        return db.get(models.User, user_id)
    if username:
        stmt = select(models.User).where(models.User.username == username)
        return db.scalars(stmt).first()
    return None

def get_users(db: Session, skip: int = 0, limit: int = 100):
    query = select(models.User).offset(skip).limit(limit)
    return db.execute(query).scalars().all()

def delete_user(db: Session, user_id: int):
    try:
        db.execute(delete(models.User).where(models.User.id == user_id))
        db.commit()
    except SQLAlchemyError as e:
        logger.error(f"Ошибка при удалении пользователя: {e}")
        db.rollback()

def add_jazz_standard(db: Session, title: str, composer: str):
    db_jazz_standard = models.JazzStandard(title=title, composer=composer)
    db.add(db_jazz_standard)
    db.commit()
    db.refresh(db_jazz_standard)
    return db_jazz_standard

def get_jazz_standard(db: Session, jazz_standard_id: int = None, title: str = None):
    query = select(models.JazzStandard)
    if jazz_standard_id:
        return db.get(models.JazzStandard, jazz_standard_id)
    if title:
        return db.execute(query.where(models.JazzStandard.title == title)).scalars().first()
    return None

def get_jazz_standards(db: Session, skip: int = 0, limit: int = 100):
    query = select(models.JazzStandard).offset(skip).limit(limit)
    return db.execute(query).scalars().all()

def delete_jazz_standard(db: Session, jazz_standard_id: int):
    try:
        db.execute(delete(models.JazzStandard).where(models.JazzStandard.id == jazz_standard_id))
        db.commit()
    except SQLAlchemyError as e:
        logger.error(f"Ошибка при удалении стандарта: {e}")
        db.rollback()

def add_standard_to_user(db: Session, user_id: int, jazz_standard_id: int):
    db_user_standard = models.UserJazzStandard(user_id=user_id, jazz_standard_id=jazz_standard_id)
    db.add(db_user_standard)
    db.commit()
    db.refresh(db_user_standard)
    return db_user_standard

def user_knows_standard(db: Session, user_id: int, jazz_standard_id: int):
    query = select(models.UserJazzStandard).where(models.UserJazzStandard.user_id == user_id).where(models.UserJazzStandard.jazz_standard_id == jazz_standard_id)
    return db.execute(query).scalars().first()

def get_user_standards(db: Session, user_id: int):
    query = select(models.UserJazzStandard).where(models.UserJazzStandard.user_id == user_id)
    return db.execute(query).scalars().all()

def delete_user_standard(db: Session, user_id: int, jazz_standard_id: int):
    try:
        db.execute(delete(models.UserJazzStandard).where(models.UserJazzStandard.user_id == user_id).where(models.UserJazzStandard.jazz_standard_id == jazz_standard_id))
        db.commit()
    except SQLAlchemyError as e:
        logger.error(f"Ошибка при удалении стандарта у пользователя: {e}")
        db.rollback()

def get_user_standards_count(db: Session, user_id: int):
    query = select(func.count(models.UserJazzStandard.jazz_standard_id)).where(models.UserJazzStandard.user_id == user_id)
    return db.execute(query).scalar()