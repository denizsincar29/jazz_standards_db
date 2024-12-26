import os
from sqlalchemy import create_engine
from sqlalchemy.orm import DeclarativeBase, sessionmaker
import logging
logging.basicConfig(level=logging.INFO)


url_start = "postgresql+psycopg://jazz:jazz@localhost/"  # use psycopg3! it's faster and better
# if there is USE_SQLITE env variable set to "1", use sqlite db
if os.environ.get("USE_SQLITE") == "1":
    url_start = "sqlite:///"
else:
    if os.environ.get("IN_DOCKER"):  # hostname is db
        logging.info("Using db hostname with docker")
        url_start.replace("localhost", "db")
# if there is test environment variable set to "1" or "2", use db test1 or test2
if os.environ.get("TEST_ENV") == "1":
    SQLALCHEMY_DATABASE_URL = f"{url_start}test1"
elif os.environ.get("TEST_ENV") == "2":
    SQLALCHEMY_DATABASE_URL = f"{url_start}test2"
else:
    SQLALCHEMY_DATABASE_URL = f"{url_start}jazz"  # use jazz db
if url_start == "sqlite:///":  # if sqlite, add .db extension
    SQLALCHEMY_DATABASE_URL += ".db"


# engine creation
# test environment will have echo on
engine = create_engine(SQLALCHEMY_DATABASE_URL, echo = (int(os.environ.get("DEBUG", 0)) > 0))

# session factory creation
Session = sessionmaker(autocommit=False, autoflush=False, bind=engine)

# base class for models. It will autoget children models when creating tables
class Base(DeclarativeBase):
    pass

# this function is used to create all tables in the database, not only in the test database
def init_db():
    from db import models  # noqa
    # if test environment, drop all tables and create them again or unique constraint will fail
    if os.environ.get("TEST_ENV") == "1" or os.environ.get("TEST_ENV") == "2":
        logging.info("Dropping all tables in test environment")
        Base.metadata.drop_all(bind=engine)
    Base.metadata.create_all(bind=engine)

# This function is only used in tests! It drops all tables in the database and closes the connection
def teardown_db():
    Base.metadata.drop_all(bind=engine)
    engine.dispose()

# generator function to get a database session and close it after use
def get_db():
    db = Session()
    try:
        yield db
    finally:
        db.close()
