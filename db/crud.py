from sqlalchemy.orm import Session
from sqlalchemy import select
from sqlalchemy import func
from sqlalchemy.exc import SQLAlchemyError, IntegrityError
import logging

from db import models

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__) # Corrected logger name


# all password hashing and salting and other stuff is not here, here we just have crud functions!
# All operations here don't check if user is admin or not, this should be done in the overlying code
def create_user(db: Session, name: str, username: str, password_hash: bytes, salt: bytes, is_admin: bool = False):
    if not name or not username:
        raise ValueError("Both 'name' and 'username' are required to create a user.")
    db_user = models.User(username=username, name=name, is_admin=is_admin)
    try:
        db.add(db_user)
        db.commit()
        db.refresh(db_user)
        db_user_private = models.UserPrivate(user_id=db_user.id, password_hash=password_hash, salt=salt)
        db.add(db_user_private)
        db.commit()
        db.refresh(db_user_private)
    except IntegrityError as e:
        logger.error(f"Error creating user: {e}")
        db.rollback()
        return None
    return db_user

def get_user(db: Session, user_id: int = None, username: str = None):
    if user_id is None and username is None:
        raise ValueError("Either 'user_id' or 'username' must be provided.")
    if user_id:
        return db.get(models.User, user_id)
    if username:
        stmt = select(models.User).where(models.User.username == username)
        return db.execute(stmt).scalars().first()

def get_users(db: Session, skip: int = 0, limit: int = 100):
    query = select(models.User).offset(skip).limit(limit)
    return db.execute(query).scalars().all()

def get_users_count(db: Session):
    return db.execute(select(func.count(models.User.id))).scalar()

def get_user_by_token(db: Session, token: str):
    query = select(models.User).join(models.UserPrivate).where(models.UserPrivate.token == token)
    return db.execute(query).scalars().first()

def update_user_token(db: Session, user_id: int = None, username: str = None, token: str = None):
    if not token:
        raise ValueError("Token is required to update user token.")
    if user_id is None and username is None:
        raise ValueError("Either 'user_id' or 'username' must be provided.")
    if user_id:
        query = select(models.UserPrivate).where(models.UserPrivate.user_id == user_id)
    elif username:
        query = select(models.UserPrivate).join(models.User).where(models.User.username == username)
    db_user_private = db.execute(query).scalars().first()
    if db_user_private:
        db_user_private.token = token
        db.commit()
        return True
    return False

def get_salt(db: Session, username: str = None, user_id: int = None):
    if username is None and user_id is None:
        raise ValueError("Either 'username' or 'user_id' must be provided.")
    if user_id:
        query = select(models.UserPrivate).where(models.UserPrivate.user_id == user_id)
    elif username:
        query = select(models.UserPrivate).join(models.User).where(models.User.username == username)
    result = db.execute(query).scalars().first()
    return result.salt if result else None #Handle case where no user is found


def check_password(db: Session, username: str, password_hash: bytes):
    if not username or not password_hash:
        raise ValueError("Both 'username' and 'password_hash' are required.")
    query = select(models.UserPrivate).join(models.User).where(models.User.username == username)
    db_user_private = db.execute(query).scalars().first()
    if db_user_private: #Check if user exists before comparing password
        return db_user_private.password_hash == password_hash
    return False

def delete_user(db: Session, user_id: int = None, username: str = None):
    if user_id is None and username is None:
        raise ValueError("Either 'user_id' or 'username' must be provided.")
    try:
        # here something is wrong, we should check if user exists before deleting
        logger.info(f"Deleting user with id {user_id} or username {username}")
        user = get_user(db, user_id=user_id, username=username)
        if not user:
            logger.info(f"User with id {user_id} or username {username} not found.") #Improved logging message
            return False
        db.delete(user)
        db.commit()
        return True
    except SQLAlchemyError as e:
        logger.error(f"Error deleting user: {e}")
        db.rollback()
        return False


def add_jazz_standard(db: Session, title: str, composer: str, style: models.JazzStyle | str):
    if not title or not composer or not style:
        raise ValueError("Title, composer, and style are all required to add a jazz standard.")
    if isinstance(style, str):
        style = models.JazzStyle.str_to_enum(style)
    db_jazz_standard = models.JazzStandard(title=title, composer=composer, style=style)
    try:
        db.add(db_jazz_standard)
        db.commit()
        db.refresh(db_jazz_standard) # Refresh after commit to get the ID
    except IntegrityError as e:
        logger.error(f"Error adding jazz standard: {e}")
        db.rollback()
        return None
    return db_jazz_standard


def get_jazz_standard(db: Session, jazz_standard_id: int = None, title: str = None):
    if jazz_standard_id is None and title is None:
        raise ValueError("Either jazz_standard_id or title must be provided.")
    if jazz_standard_id:
        return db.get(models.JazzStandard, jazz_standard_id)
    elif title:
        query = select(models.JazzStandard).where(models.JazzStandard.title == title)
        return db.execute(query).scalars().first()


def search_jazz_standard(db: Session, search: str):
    if not search:
        raise ValueError("Search term cannot be empty.") #added validation
    query = select(models.JazzStandard).where(models.JazzStandard.title.ilike(f"%{search}%"))
    return db.execute(query).scalars().all()


def get_standards_by_composer(db: Session, composer: str):
    if not composer:
        raise ValueError("Composer name cannot be empty.") #added validation
    query = select(models.JazzStandard).where(models.JazzStandard.composer.ilike(f"%{composer}%"))
    return db.execute(query).scalars().all()


def get_standards_by_style(db: Session, style: models.JazzStyle | str):
    if not style:
        raise ValueError("Style cannot be None.") #added validation
    if isinstance(style, str):
        style = models.JazzStyle.str_to_enum(style)
    query = select(models.JazzStandard).where(models.JazzStandard.style == style)
    return db.execute(query).scalars().all()


def get_jazz_standards(db: Session, skip: int = 0, limit: int = 100):
    query = select(models.JazzStandard).offset(skip).limit(limit)
    return db.execute(query).scalars().all()


def delete_jazz_standard(db: Session, jazz_standard_id: int = None, title: str = None):
    if jazz_standard_id is None and title is None:
        raise ValueError("Either jazz_standard_id or title must be provided.")
    try:
        standard = get_jazz_standard(db, jazz_standard_id=jazz_standard_id, title=title)
        if not standard:
            logger.info(f"Jazz standard with id {jazz_standard_id or title} not found.")
            return False
        db.delete(standard)
        db.commit()
        return True
    except SQLAlchemyError as e:
        logger.error(f"Error deleting jazz standard: {e}")
        db.rollback()
        return False

def add_standard_to_user(db: Session, user_id: int = None, username: str = None, jazz_standard_id: int = None, title: str = None):
    if user_id is None and username is None:
        raise ValueError("Either user_id or username must be provided.")
    if jazz_standard_id is None and title is None:
        raise ValueError("Either jazz_standard_id or title must be provided.")
    if user_knows_standard(db, user_id=user_id, username=username, jazz_standard_id=jazz_standard_id, title=title):
        logger.info(f"User already knows standard with id {jazz_standard_id or title}") #Improved logging message
        return None

    try:
        user_id_to_use = user_id or get_user_id_by_username(db, username) #Helper function - see below
        db_user_standard = models.UserJazzStandard(user_id=user_id_to_use, jazz_standard_id=jazz_standard_id or get_jazz_standard_id_by_title(db, title)) #Helper function - see below
        db.add(db_user_standard)
        db.commit()
        db.refresh(db_user_standard)
        return db_user_standard
    except (ValueError, SQLAlchemyError) as e: #Catch both ValueError and SQLAlchemyError
        logger.error(f"Error adding standard to user: {e}")
        db.rollback()
        return None

def user_knows_standard(db: Session, user_id: int = None, username: str = None, jazz_standard_id: int = None, title: str = None):
    if user_id is None and username is None:
        raise ValueError("Either user_id or username must be provided.")
    if jazz_standard_id is None and title is None:
        raise ValueError("Either jazz_standard_id or title must be provided.")

    query = select(models.UserJazzStandard)
    user_id_to_use = user_id or get_user_id_by_username(db, username) #Helper function - see below
    standard_id_to_use = jazz_standard_id or get_jazz_standard_id_by_title(db, title) #Helper function - see below

    query = query.where(models.UserJazzStandard.user_id == user_id_to_use, models.UserJazzStandard.jazz_standard_id == standard_id_to_use)
    return db.execute(query).scalars().first()


def get_user_standards(db: Session, user_id: int = None, username: str = None):
    if user_id is None and username is None:
        raise ValueError("Either user_id or username must be provided.")
    query = select(models.UserJazzStandard)
    if user_id:
        query = query.where(models.UserJazzStandard.user_id == user_id)
    elif username:
        query = query.join(models.User).where(models.User.username == username)
    return db.execute(query).scalars().all()


def delete_user_standard(db: Session, user_id: int = None, username: str = None, jazz_standard_id: int = None, title: str = None):
    if user_id is None and username is None:
        raise ValueError("Either user_id or username must be provided.")
    if jazz_standard_id is None and title is None:
        raise ValueError("Either jazz_standard_id or title must be provided.")

    try:
        # logic as delete user:
        user_standard = user_knows_standard(db, user_id=user_id, username=username, jazz_standard_id=jazz_standard_id, title=title)
        if not user_standard:
            logger.info(f"User does not know standard with id {jazz_standard_id or title}")
            return False
        db.delete(user_standard)
        db.commit()
        return True
    except (ValueError, SQLAlchemyError) as e: #Catch both ValueError and SQLAlchemyError
        logger.error(f"Error deleting user standard: {e}")
        db.rollback()
        return False

def get_user_standards_count(db: Session, user_id: int = None, username: str = None):
    if user_id is None and username is None:
        raise ValueError("Either user_id or username must be provided.")
    query = select(func.count(models.UserJazzStandard.jazz_standard_id))
    if user_id:
        query = query.where(models.UserJazzStandard.user_id == user_id)
    elif username:
        query = query.join(models.User).where(models.User.username == username)
    return db.execute(query).scalar()

def get_user_id_by_username(db: Session, username: str):
    if not username:
        raise ValueError("Username cannot be empty")
    user = db.execute(select(models.User).where(models.User.username == username)).scalar()
    if user:
        return user.id
    raise ValueError(f"User with username '{username}' not found.")

def get_jazz_standard_id_by_title(db: Session, title: str):
    if not title:
        raise ValueError("Title cannot be empty")
    standard = db.execute(select(models.JazzStandard).where(models.JazzStandard.title == title)).scalar()
    if standard:
        return standard.id
    raise ValueError(f"Jazz standard with title '{title}' not found.")