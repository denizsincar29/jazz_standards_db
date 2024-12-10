from datetime import datetime

from sqlalchemy.orm import Session
from sqlalchemy import select, update, delete
from sqlalchemy import func
from sqlalchemy.exc import SQLAlchemyError, IntegrityError
import logging

from db import models

# Настройка логирования
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def create_user(db: Session, username: str, name: str, is_admin: bool = False):
    db_user = models.User(username=username, name=name, is_admin=is_admin)
    try:
        db.add(db_user)
        db.commit()
    except IntegrityError as e:
        logger.error(f"Ошибка при создании пользователя: {e}")
        db.rollback()
        return None
    db.refresh(db_user)
    return db_user

def get_user(db: Session, user_id: int = None, username: str = None):
    if user_id:
        return db.get(models.User, user_id)
    if username:
        stmt = select(models.User).where(models.User.username == username)
        return db.execute(stmt).scalars().first()
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

def add_jazz_standard(db: Session, title: str, composer: str, style: models.JazzStyle):
    db_jazz_standard = models.JazzStandard(title=title, composer=composer, style=style)
    try:
        db.add(db_jazz_standard)
        db.commit()
    except IntegrityError as e:
        logger.error(f"Ошибка при добавлении стандарта: {e}")
        db.rollback()
        return None
    db.refresh(db_jazz_standard)
    return db_jazz_standard

def get_jazz_standard(db: Session, jazz_standard_id: int = None, title: str = None):
    query = select(models.JazzStandard)
    if jazz_standard_id:
        return db.get(models.JazzStandard, jazz_standard_id)
    if title:
        return db.execute(query.where(models.JazzStandard.title == title)).scalars().first()
    return None

def search_jazz_standard(db: Session, search: str):
    query = select(models.JazzStandard).where(models.JazzStandard.title.ilike(f"%{search}%"))
    return db.execute(query).scalars().all()

def get_standards_by_composer(db: Session, composer: str):
    query = select(models.JazzStandard).where(models.JazzStandard.composer.ilike(f"%{composer}%"))
    return db.execute(query).scalars().all()

def get_standards_by_style(db: Session, style: models.JazzStyle):
    query = select(models.JazzStandard).where(models.JazzStandard.style == style)
    return db.execute(query).scalars().all()

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
    # check if user already knows the standard
    if user_knows_standard(db, user_id, jazz_standard_id):
        logger.info(f"Пользователь уже знает стандарт с id {jazz_standard_id}")
        return None
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