import os
try:
    os.remove("test.db")  # remove the test db file if it exists
except FileNotFoundError:
    pass
os.environ["JAZZ_DB_FILE"] = "test.db"  # because :memory: is doing some weird stuff
from fastapi.testclient import TestClient #noqa
from app import app #noqa

client = TestClient(app)

def test_read_root():
    response = client.get("/api/")
    assert response.status_code == 200
    assert response.json() == {"users": [], "jazz_standards": []}

def test_create_user():
    response = client.post(
        "/api/users/",
        json={"username": "test_user", "name": "Test User", "is_admin": True}
    )
    assert response.status_code == 200
    assert response.json() == {"id": 1, "username": "test_user", "name": "Test User", "is_admin": True}

def test_read_user():
    response = client.post(
        "/api/users/",
        json={"username": "other_user", "name": "Other User", "is_admin": False}
    )
    user_id = response.json()["id"]
    response = client.get(f"/api/users/{user_id}")
    assert response.status_code == 200
    assert response.json() == {"id": user_id, "username": "other_user", "name": "Other User", "is_admin": False}

def test_read_users():
    response = client.post(
        "/api/users/",
        json={"username": "test_user1", "name": "Test User 1", "is_admin": True}
    )
    response = client.post(
        "/api/users/",
        json={"username": "test_user2", "name": "Test User 2", "is_admin": False}
    )
    response = client.get("/api/users/")
    assert response.status_code == 200
    # i think we have the previous users from the previous tests, so lets just check with "in"
    j = response.json()
    assert {"id": 3, "username": "test_user1", "name": "Test User 1", "is_admin": True} in j
    assert {"id": 4, "username": "test_user2", "name": "Test User 2", "is_admin": False} in j

def test_delete_user():
    response = client.post(
        "/api/users/",
        json={"username": "delete_user", "name": "Delete User", "is_admin": False}
    )
    j = response.json()
    print(j)
    user_id = j["id"]
    response = client.delete(f"/api/users/{user_id}")
    assert response.status_code == 200
    assert response.json() == {"id": user_id, "username": "delete_user", "name": "Delete User", "is_admin": False}
    # now lets delete non existing user
    response = client.delete(f"/api/users/{user_id}")
    assert response.status_code == 404  # yikes!
