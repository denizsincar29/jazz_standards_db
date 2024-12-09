import pytest
import os
os.environ["JAZZ_DB_FILE"] = "jazz_test.db"


from db import models, crud, SessionLocal, init_db

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

    def test_get_user(self, db):
        user = crud.create_user(db, "test_user", "Test User")
        user_id = user.id
        user = crud.get_user(db, user_id=user_id)
        assert isinstance(user, models.User)
        assert user.username == "test_user"
        assert user.name == "Test User"

    def test_get_user_by_username(self, db):
        user = crud.create_user(db, "test_user", "Test User")
        user = crud.get_user(db, username="test_user")
        assert isinstance(user, models.User)
        assert user.username == "test_user"
        assert user.name == "Test User"

    def test_get_users(self, db):
        crud.create_user(db, "test_user1", "Test User 1")
        crud.create_user(db, "test_user2", "Test User 2")
        users = crud.get_users(db)
        assert isinstance(users, list)
        assert isinstance(users[0], models.User)
        assert isinstance(users[1], models.User)

    def test_delete_user(self, db):
        user = crud.create_user(db, "test_user", "Test User")
        user_id = user.id
        crud.delete_user(db, user_id)
        user = crud.get_user(db, user_id=user_id)
        assert user is None

    def test_add_jazz_standard(self, db):
        jazz_standard = crud.add_jazz_standard(db, "Autumn Leaves", "Joseph Kosma")  # copilot doesn't know other jazz standards than Autumn Leaves xD
        assert isinstance(jazz_standard, models.JazzStandard)
        assert jazz_standard.title == "Autumn Leaves"
        assert jazz_standard.composer == "Joseph Kosma"

    def test_get_jazz_standard(self, db):
        jazz_standard = crud.add_jazz_standard(db, "Autumn Leaves", "Joseph Kosma")
        jazz_standard_id = jazz_standard.id
        jazz_standard = crud.get_jazz_standard(db, jazz_standard_id=jazz_standard_id)
        assert isinstance(jazz_standard, models.JazzStandard)
        assert jazz_standard.title == "Autumn Leaves"
        assert jazz_standard.composer == "Joseph Kosma"

    def test_get_jazz_standard_by_title(self, db):
        jazz_standard = crud.add_jazz_standard(db, "Autumn Leaves", "Joseph Kosma")
        jazz_standard = crud.get_jazz_standard(db, title="Autumn Leaves")
        assert isinstance(jazz_standard, models.JazzStandard)
        assert jazz_standard.title == "Autumn Leaves"
        assert jazz_standard.composer == "Joseph Kosma"

    def test_get_jazz_standards(self, db):
        crud.add_jazz_standard(db, "Autumn Leaves", "Joseph Kosma")
        crud.add_jazz_standard(db, "Blue Bossa", "Kenny Dorham")
        jazz_standards = crud.get_jazz_standards(db)
        assert isinstance(jazz_standards, list)
        assert isinstance(jazz_standards[0], models.JazzStandard)
        assert isinstance(jazz_standards[1], models.JazzStandard)

    def test_delete_jazz_standard(self, db):
        jazz_standard = crud.add_jazz_standard(db, "Autumn Leaves", "Joseph Kosma")
        jazz_standard_id = jazz_standard.id
        crud.delete_jazz_standard(db, jazz_standard_id)
        jazz_standard = crud.get_jazz_standard(db, jazz_standard_id=jazz_standard_id)
        assert jazz_standard is None

    def test_add_standard_to_user(self, db):
        user2 = crud.create_user(db, "test_user2", "Test User 2")
        jazz_standard = crud.add_jazz_standard(db, "Caravan", "Duke Ellington")
        # add jazz standard to user
        crud.add_standard_to_user(db, user2.id, jazz_standard.id)
        user2 = crud.get_user(db, user2.id)
        print(user2.jazz_standards)
        assert jazz_standard == user2.jazz_standards[0].jazz_standard