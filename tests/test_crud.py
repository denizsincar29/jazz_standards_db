import pytest
import os
import bcrypt
from sqlalchemy import select, delete
os.environ["TEST_ENV"] ="1"
from db import models, crud, Session, engine, init_db, JazzStyle  # noqa  # linter doesn't like imports after setting env vars
import logging
logging.basicConfig(level=logging.DEBUG)  # so we can see the queries


password = b"password"
salt = bcrypt.gensalt()
hashed_password = bcrypt.hashpw(password, salt)

# This thing is used for getting sessions.
@pytest.fixture()
def db():
    logging.debug("Creating a new session")
    db = Session()
    yield db
    db.close()
    logging.debug("Closing the session")

@pytest.fixture(scope="module", autouse=True)  # autouse means it will be used by all tests
def setup():
    logging.debug("Setting up the database")
    init_db()
    yield  # this is where the tests will run
    engine.dispose()
    logging.debug("Disposing the engine")


# tests are run one by one, and all entries stay in the db, so everytime we make a user with the same username, it will fail
class TestCRUD:
    def test_create_user(self, db):
        user = crud.create_user(db, name="Test User", username="testuser", password_hash=hashed_password, salt=salt)  # salt is not nullable
        assert user
        assert user.name == "Test User"
        assert user.username == "testuser"
        assert user.id is not None

    def test_check_password(self, db):
        crud.create_user(db, name="Test User", username="testuser_a", password_hash=hashed_password, salt=salt)
        assert crud.check_password(db, "testuser_a", hashed_password)
        assert not crud.check_password(db, "testuser_a", b"wrongpassword")

    def test_get_user(self, db):
        user = crud.create_user(db, name="Test User", username="testuser_b", password_hash=hashed_password, salt=salt)
        retrieved_user = crud.get_user(db, user_id=user.id)
        assert retrieved_user == user

    def test_get_user_by_username(self, db):
        user = crud.create_user(db, name="Test User", username="testuser_c", password_hash=hashed_password, salt=salt)
        retrieved_user = crud.get_user(db, username="testuser_c")
        assert retrieved_user == user


    def test_get_users(self, db):
        crud.create_user(db, name="Test User 1", username="testuser1", password_hash=hashed_password, salt=salt)
        crud.create_user(db, name="Test User 2", username="testuser2", password_hash=hashed_password, salt=salt)
        users = crud.get_users(db)
        assert len(users) >= 2  # i think previous tests might have added some users

    def test_delete_user(self, db):
        user = crud.create_user(db, name="Test User", username="testuser_d", password_hash=hashed_password, salt=salt)
        assert crud.delete_user(db, user_id=user.id)
        retrieved_user = crud.get_user(db, user_id=user.id)
        assert retrieved_user is None
        # lets retrieve the private data of the user because i think it has a bug and it doesn't delete the private data
        private_data = db.execute(select(models.UserPrivate).where(models.UserPrivate.user_id == user.id)).scalar_one_or_none()
        assert private_data is None  # it should be None because we deleted the user
        # it will not raise an error if the user is not found, it will just delete nothing

    def test_add_jazz_standard(self, db):
        style = JazzStyle.modal  # so what is modal jazz
        standard = crud.add_jazz_standard(db, title="So What", composer="Miles Davis", style=style)
        assert standard
        assert standard.title == "So What"
        assert standard.composer == "Miles Davis"
        assert standard.style == style
        assert standard.id is not None
        result = crud.add_jazz_standard(db, title="So What", composer="Miles Davis", style=style)  # Boom!
        assert result is None  # it will return None if the standard already exists

    def test_get_jazz_standard(self, db):
        style = JazzStyle.bebop
        standard = crud.add_jazz_standard(db, title = "Donna Lee", composer = "Charlie Parker", style = style)
        retrieved_standard = crud.get_jazz_standard(db, jazz_standard_id=standard.id)
        assert retrieved_standard == standard

    def test_get_jazz_standard_by_title(self, db):
        style = JazzStyle.bebop  # bebop by dizzy gillespie
        standard = crud.add_jazz_standard(db, title="Salt Peanuts", composer="Dizzy Gillespie", style=style)
        retrieved_standard = crud.get_jazz_standard(db, title="Salt Peanuts")
        assert retrieved_standard == standard


    def test_add_standard_to_user(self, db):
        user = crud.create_user(db, name="Test User", username="testuser_footprint", password_hash=hashed_password, salt=salt)
        style = JazzStyle.modal
        standard = crud.add_jazz_standard(db, title="Footprints", composer="Wayne Shorter", style=style)
        user_standard = crud.add_standard_to_user(db, user_id=user.id, jazz_standard_id=standard.id)
        assert user_standard
        assert user_standard.user_id == user.id
        assert user_standard.jazz_standard_id == standard.id


    def test_user_knows_standard(self, db):
        user = crud.create_user(db, name="Test User", username="testuser_someone", password_hash=hashed_password, salt=salt)
        style = JazzStyle.bebop
        standard = crud.add_jazz_standard(db, title="Four", composer="Miles Davis", style=style)
        crud.add_standard_to_user(db, user_id=user.id, jazz_standard_id=standard.id)
        knows_standard = crud.user_knows_standard(db, user_id=user.id, jazz_standard_id=standard.id)
        assert knows_standard

    def test_get_user_standards(self, db):
        user = crud.create_user(db, name="Test User", username="testuser_f", password_hash=hashed_password, salt=salt)
        style = JazzStyle.bossa_nova
        standard = crud.add_jazz_standard(db, title="Blue Bossa", composer="Kenny Dorham", style=style)
        crud.add_standard_to_user(db, user_id=user.id, jazz_standard_id=standard.id)
        user_standards = crud.get_user_standards(db, user_id=user.id)
        assert len(user_standards) == 1  # here we have only one standard because previous tests used different users


    def test_delete_user_standard(self, db):
        user = crud.create_user(db, name="Test User", username="testuser_g", password_hash=hashed_password, salt=salt)
        style = JazzStyle.bebop
        standard = crud.add_jazz_standard(db, title="Ormithology", composer="Charlie Parker", style=style)
        crud.add_standard_to_user(db, user_id=user.id, jazz_standard_id=standard.id)
        assert crud.delete_user_standard(db, user_id=user.id, jazz_standard_id=standard.id)
        user_standards = crud.get_user_standards(db, user_id=user.id)
        assert len(user_standards) == 0

    def test_get_user_standards_count(self, db):
        user = crud.create_user(db, name="Test User", username="testuser_h", password_hash=hashed_password, salt=salt)
        style = JazzStyle.bebop
        standard = crud.add_jazz_standard(db, title="Anthropology", composer="Charlie Parker", style=style)
        crud.add_standard_to_user(db, user_id=user.id, jazz_standard_id=standard.id)
        count = crud.get_user_standards_count(db, user_id=user.id)
        assert count == 1

    def tests_delete_everything(self, db):
        # crud doesn't have a delete_everything function, so we'll just delete everything manually
        db.execute(delete(models.User))
        db.execute(delete(models.JazzStandard))
        #db.execute(delete(models.UserPrivate))  # we don't need to delete private data because it will be deleted with the user
        #db.execute(delete(models.UserJazzStandard))  # we don't need to delete user standards because they will be deleted with the user
        db.commit()
        # lets check if everything is deleted
        assert len(crud.get_users(db)) == 0
        assert len(crud.get_jazz_standards(db)) == 0
        # lets check if private data and user standards are deleted
        assert db.execute(select(models.UserPrivate)).scalar_one_or_none() is None
        assert db.execute(select(models.UserJazzStandard)).scalar_one_or_none() is None
        # good bye all, it was nice testing you