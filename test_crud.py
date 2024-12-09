import unittest

from db import models, crud, SessionLocal, init_db

class TestCRUD(unittest.TestCase):
    def setUp(self):
        self.db = SessionLocal()
        init_db()

    def tearDown(self):
        self.db.close()

    def test_create_user(self):
        user = crud.create_user(self.db, "test_user", "Test User")
        self.assertIsInstance(user, models.User)
        self.assertEqual(user.username, "test_user")
        self.assertEqual(user.name, "Test User")

    def test_get_user(self):
        user = crud.create_user(self.db, "test_user", "Test User")
        user_id = user.id
        user = crud.get_user(self.db, user_id=user_id)
        self.assertIsInstance(user, models.User)
        self.assertEqual(user.username, "test_user")
        self.assertEqual(user.name, "Test User")

    def test_get_user_by_username(self):
        user = crud.create_user(self.db, "test_user", "Test User")
        user = crud.get_user(self.db, username="test_user")
        self.assertIsInstance(user, models.User)
        self.assertEqual(user.username, "test_user")
        self.assertEqual(user.name, "Test User")

    def test_get_users(self):
        crud.create_user(self.db, "test_user1", "Test User 1")
        crud.create_user(self.db, "test_user2", "Test User 2")
        users = crud.get_users(self.db)
        self.assertIsInstance(users, list)
        self.assertIsInstance(users[0], models.User)
        self.assertIsInstance(users[1], models.User)

    def test_delete_user(self):
        user = crud.create_user(self.db, "test_user", "Test User")
        user_id = user.id
        crud.delete_user(self.db, user_id)
        user = crud.get_user(self.db, user_id=user_id)
        self.assertIsNone(user)

    def test_add_jazz_standard(self):
        jazz_standard = crud.add_jazz_standard(self.db, "Autumn Leaves", "Joseph Kosma")  # copilot doesn't know other jazz standards than Autumn Leaves xD
        self.assertIsInstance(jazz_standard, models.JazzStandard)
        self.assertEqual(jazz_standard.title, "Autumn Leaves")
        self.assertEqual(jazz_standard.composer, "Joseph Kosma")

    def test_get_jazz_standard(self):
        jazz_standard = crud.add_jazz_standard(self.db, "Autumn Leaves", "Joseph Kosma")
        jazz_standard_id = jazz_standard.id
        jazz_standard = crud.get_jazz_standard(self.db, jazz_standard_id=jazz_standard_id)
        self.assertIsInstance(jazz_standard, models.JazzStandard)
        self.assertEqual(jazz_standard.title, "Autumn Leaves")
        self.assertEqual(jazz_standard.composer, "Joseph Kosma")

    def test_get_jazz_standard_by_title(self):
        jazz_standard = crud.add_jazz_standard(self.db, "Autumn Leaves", "Joseph Kosma")
        jazz_standard = crud.get_jazz_standard(self.db, title="Autumn Leaves")
        self.assertIsInstance(jazz_standard, models.JazzStandard)
        self.assertEqual(jazz_standard.title, "Autumn Leaves")
        self.assertEqual(jazz_standard.composer, "Joseph Kosma")

    def test_get_jazz_standards(self):
        crud.add_jazz_standard(self.db, "Autumn Leaves", "Joseph Kosma")
        crud.add_jazz_standard(self.db, "Blue Bossa", "Kenny Dorham")
        jazz_standards = crud.get_jazz_standards(self.db)
        self.assertIsInstance(jazz_standards, list)
        self.assertIsInstance(jazz_standards[0], models.JazzStandard)
        self.assertIsInstance(jazz_standards[1], models.JazzStandard)

    def test_delete_jazz_standard(self):
        jazz_standard = crud.add_jazz_standard(self.db, "Autumn Leaves", "Joseph Kosma")
        jazz_standard_id = jazz_standard.id
        crud.delete_jazz_standard(self.db, jazz_standard_id)
        jazz_standard = crud.get_jazz_standard(self.db, jazz_standard_id=jazz_standard_id)
        self.assertIsNone(jazz_standard)

    def test_add_standard_to_user(self):
        user = crud.create_user(self.db, "test_user", "Test User")
        jazz_standard = crud.add_jazz_standard(self.db, "Autumn Leaves", "Joseph Kosma")
        user_id = user.id
        jazz_standard_id = jazz_standard.id
        crud.add_standard_to_user(self.db, user_id, jazz_standard_id)
        user = crud.get_user(self.db, user_id=user_id)
        self.assertIsInstance(user.jazz_standards, list)
        first_standard: models.UserJazzStandard = user.jazz_standards[0] # for vscode to know the type
        self.assertIsInstance(user.jazz_standards[0], models.UserJazzStandard)
        self.assertEqual(user.jazz_standards[0].jazz_standard.title, "Autumn Leaves")
        self.assertEqual(user.jazz_standards[0].jazz_standard.composer, "Joseph Kosma")

