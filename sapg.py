import os
from sqlalchemy import select
os.environ["TEST_ENV"] ="1"
from db import models, crud, Session, engine, init_db, JazzStyle  # noqa  # linter doesn't like imports after setting env vars


def db():
    init_db()
    db = Session()
    yield db
    db.close()

# this is an importable playground for testing
