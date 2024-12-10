import pytest
import os
os.environ["JAZZ_DB_FILE"] = ":memory:"  # Use in-memory database for testing
from db import models, crud, SessionLocal, init_db  #noqa  # ruff warns me that not at the top of the file

# This thing is used for getting sessions.
@pytest.fixture()
def db():
    db = SessionLocal()
    init_db()
    yield db
    db.close()

class TestCRUD:
    def test_create_user(self, db):
        user = crud.create_user(db, "test_user", "Test User")
        assert isinstance(user, models.User)
        assert user.username == "test_user"
        assert user.name == "Test User"
        # create it again, should return None
        user = crud.create_user(db, "test_user", "Test User")
        assert user is None  # yikes!

    def test_get_user(self, db):
        user = crud.create_user(db, "other_user", "Other User")
        user_id = user.id
        user = crud.get_user(db, user_id=user_id)
        assert isinstance(user, models.User)
        assert user.username == "other_user"
        assert user.name == "Other User"

    def test_get_user_by_username(self, db):
        user = crud.create_user(db, "yet_another_user", "Yet Another User")
        user = crud.get_user(db, username="yet_another_user")
        assert isinstance(user, models.User)
        assert user.username == "yet_another_user"
        assert user.name == "Yet Another User"

    def test_get_users(self, db):
        crud.create_user(db, "test_user1", "Test User 1")
        crud.create_user(db, "test_user2", "Test User 2")
        users = crud.get_users(db)
        assert isinstance(users, list)
        assert isinstance(users[0], models.User)
        assert isinstance(users[1], models.User)

    def test_delete_user(self, db):
        user = crud.create_user(db, "test_user3", "Test User 3")
        user_id = user.id
        crud.delete_user(db, user_id)
        user = crud.get_user(db, user_id=user_id)
        assert user is None

    def test_add_jazz_standard(self, db):
        jazz_standard = crud.add_jazz_standard(db, "Autumn Leaves", "Joseph Kosma", models.JazzStyle.swing)
        assert isinstance(jazz_standard, models.JazzStandard)
        assert jazz_standard.title == "Autumn Leaves"
        assert jazz_standard.composer == "Joseph Kosma"

    def test_get_jazz_standard(self, db):
        jazz_standard = crud.add_jazz_standard(db, "Spain", "Chick Corea", models.JazzStyle.latin)
        jazz_standard_id = jazz_standard.id
        jazz_standard = crud.get_jazz_standard(db, jazz_standard_id=jazz_standard_id)
        assert isinstance(jazz_standard, models.JazzStandard)
        assert jazz_standard.title == "Spain"
        assert jazz_standard.composer == "Chick Corea"
        assert jazz_standard.style == models.JazzStyle.latin

    def test_get_jazz_standard_by_title(self, db):
        jazz_standard = crud.get_jazz_standard(db, title="Autumn Leaves")  # added in test_add_jazz_standard
        assert isinstance(jazz_standard, models.JazzStandard)
        assert jazz_standard.title == "Autumn Leaves"
        assert jazz_standard.composer == "Joseph Kosma"

    def test_get_jazz_standards(self, db):
        crud.add_jazz_standard(db, "Blue Bossa", "Kenny Dorham", models.JazzStyle.bossa_nova)
        jazz_standards = crud.get_jazz_standards(db)
        assert isinstance(jazz_standards, list)
        assert isinstance(jazz_standards[0], models.JazzStandard)
        assert isinstance(jazz_standards[1], models.JazzStandard)

    def test_delete_jazz_standard(self, db):
        jazz_standard = crud.add_jazz_standard(db, "Caravan", "Duke Ellington", models.JazzStyle.latin_swing)
        jazz_standard_id = jazz_standard.id
        crud.delete_jazz_standard(db, jazz_standard_id)
        jazz_standard = crud.get_jazz_standard(db, jazz_standard_id=jazz_standard_id)
        assert jazz_standard is None
        # delete a jazz standard that does not exist
        r = crud.delete_jazz_standard(db, jazz_standard_id=999)
        assert r is None

    def test_add_standard_to_user(self, db):
        user2 = crud.create_user(db, "user2", "User 2")
        jazz_standard = crud.add_jazz_standard(db, "All the Things You Are", "Jerome Kern", models.JazzStyle.swing)
        # add jazz standard to user
        crud.add_standard_to_user(db, user2.id, jazz_standard.id)
        user2 = crud.get_user(db, user2.id)
        print(user2.jazz_standards)
        assert jazz_standard == user2.jazz_standards[0].jazz_standard