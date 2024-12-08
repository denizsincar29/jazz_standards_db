import unittest
from unittest.mock import MagicMock
from sqlalchemy.orm import Session
from db import models, crud

class TestCRUD(unittest.TestCase):

    def setUp(self):
        self.db = MagicMock(spec=Session)

    def test_create_user(self):
        self.db.add = MagicMock()
        self.db.commit = MagicMock()
        self.db.refresh = MagicMock()
        user = crud.create_user(self.db, username="testuser", name="Test User")
        self.db.add.assert_called_once()
        self.db.commit.assert_called_once()
        self.db.refresh.assert_called_once()
        self.assertEqual(user.username, "testuser")
        self.assertEqual(user.name, "Test User")

    def test_get_user_by_id(self):
        self.db.execute = MagicMock(return_value=MagicMock(scalars=MagicMock(return_value=[models.User(id=1, username="testuser", name="Test User")])))
        user = crud.get_user(self.db, user_id=1)
        self.assertEqual(user.id, 1)
        self.assertEqual(user.username, "testuser")

    def test_get_user_by_username(self):
        self.db.execute = MagicMock(return_value=MagicMock(scalars=MagicMock(return_value=[models.User(id=1, username="testuser", name="Test User")])))
        user = crud.get_user(self.db, username="testuser")
        self.assertEqual(user.username, "testuser")
        self.assertEqual(user.name, "Test User")

    def test_get_users(self):
        self.db.execute = MagicMock(return_value=MagicMock(scalars=MagicMock(return_value=[models.User(id=1, username="testuser", name="Test User")])))
        users = crud.get_users(self.db)
        self.assertEqual(len(users), 1)
        self.assertEqual(users[0].username, "testuser")

    def test_delete_user(self):
        self.db.execute = MagicMock()
        self.db.commit = MagicMock()
        crud.delete_user(self.db, user_id=1)
        self.db.execute.assert_called_once()
        self.db.commit.assert_called_once()

    def test_add_jazz_standard(self):
        self.db.add = MagicMock()
        self.db.commit = MagicMock()
        self.db.refresh = MagicMock()
        jazz_standard = crud.add_jazz_standard(self.db, title="Autumn Leaves", composer="Joseph Kosma")
        self.db.add.assert_called_once()
        self.db.commit.assert_called_once()
        self.db.refresh.assert_called_once()
        self.assertEqual(jazz_standard.title, "Autumn Leaves")
        self.assertEqual(jazz_standard.composer, "Joseph Kosma")

    def test_get_jazz_standard_by_id(self):
        self.db.execute = MagicMock(return_value=MagicMock(scalars=MagicMock(return_value=[models.JazzStandard(id=1, title="Autumn Leaves", composer="Joseph Kosma")])))
        jazz_standard = crud.get_jazz_standard(self.db, jazz_standard_id=1)
        self.assertEqual(jazz_standard.id, 1)
        self.assertEqual(jazz_standard.title, "Autumn Leaves")

    def test_get_jazz_standard_by_title(self):
        self.db.execute = MagicMock(return_value=MagicMock(scalars=MagicMock(return_value=[models.JazzStandard(id=1, title="Autumn Leaves", composer="Joseph Kosma")])))
        jazz_standard = crud.get_jazz_standard(self.db, title="Autumn Leaves")
        self.assertEqual(jazz_standard.title, "Autumn Leaves")
        self.assertEqual(jazz_standard.composer, "Joseph Kosma")

    def test_get_jazz_standards(self):
        self.db.execute = MagicMock(return_value=MagicMock(scalars=MagicMock(return_value=[models.JazzStandard(id=1, title="Autumn Leaves", composer="Joseph Kosma")])))
        jazz_standards = crud.get_jazz_standards(self.db)
        self.assertEqual(len(jazz_standards), 1)
        self.assertEqual(jazz_standards[0].title, "Autumn Leaves")

    def test_delete_jazz_standard(self):
        self.db.execute = MagicMock()
        self.db.commit = MagicMock()
        crud.delete_jazz_standard(self.db, jazz_standard_id=1)
        self.db.execute.assert_called_once()
        self.db.commit.assert_called_once()

    def test_add_standard_to_user(self):
        self.db.add = MagicMock()
        self.db.commit = MagicMock()
        self.db.refresh = MagicMock()
        user_standard = crud.add_standard_to_user(self.db, user_id=1, jazz_standard_id=1)
        self.db.add.assert_called_once()
        self.db.commit.assert_called_once()
        self.db.refresh.assert_called_once()
        self.assertEqual(user_standard.user_id, 1)
        self.assertEqual(user_standard.jazz_standard_id, 1)

    def test_user_knows_standard(self):
        self.db.execute = MagicMock(return_value=MagicMock(scalars=MagicMock(return_value=[models.UserJazzStandard(user_id=1, jazz_standard_id=1)])))
        user_standard = crud.user_knows_standard(self.db, user_id=1, jazz_standard_id=1)
        self.assertEqual(user_standard.user_id, 1)
        self.assertEqual(user_standard.jazz_standard_id, 1)

    def test_get_user_standards(self):
        self.db.execute = MagicMock(return_value=MagicMock(scalars=MagicMock(return_value=[models.UserJazzStandard(user_id=1, jazz_standard_id=1)])))
        user_standards = crud.get_user_standards(self.db, user_id=1)
        self.assertEqual(len(user_standards), 1)
        self.assertEqual(user_standards[0].user_id, 1)

    def test_delete_user_standard(self):
        self.db.execute = MagicMock()
        self.db.commit = MagicMock()
        crud.delete_user_standard(self.db, user_id=1, jazz_standard_id=1)
        self.db.execute.assert_called_once()
        self.db.commit.assert_called_once()

    def test_get_user_standards_count(self):
        self.db.execute = MagicMock(return_value=MagicMock(scalar=MagicMock(return_value=1)))
        count = crud.get_user_standards_count(self.db, user_id=1)
        self.assertEqual(count, 1)

if __name__ == '__main__':
    unittest.main()