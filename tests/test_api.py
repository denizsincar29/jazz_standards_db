import os
os.environ["JAZZ_DB_FILE"] = ":memory:"  # Use in-memory database for testing
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
        json={"username": "test_user", "name": "Test User"}
    )
    assert response.status_code == 200
    assert response.json() == {"username": "test_user", "name": "Test User"}

def test_read_user():
    response = client.post(
        "/api/users/",
        json={"username": "other_user", "name": "Other User"}
    )
    user_id = response.json()["id"]
    response = client.get(f"/api/users/{user_id}")
    assert response.status_code == 200
    assert response.json() == {"id": user_id, "username": "other_user", "name": "Other User"}

def test_read_users():
    response = client.post(
        "/api/users/",
        json={"username": "test_user1", "name": "Test User 1"}
    )
    response = client.post(
        "/api/users/",
        json={"username": "test_user2", "name": "Test User 2"}
    )
    response = client.get("/api/users/")
    assert response.status_code == 200
    # i think we have the previous users from the previous tests, so lets just check with "in"
    j = response.json()
    assert {"username": "test_user1", "name": "Test User 1"} in j
    assert {"username": "test_user2", "name": "Test User 2"} in j

def test_delete_user():
    response = client.post(
        "/api/users/",
        json={"username": "delete_user", "name": "Delete User"}
    )
    user_id = response.json()["id"]
    response = client.delete(f"/api/users/{user_id}")
    assert response.status_code == 200
    assert response.json() == {"id": user_id, "username": "delete_user", "name": "Delete User"}
    # now lets delete non existing user
    response = client.delete(f"/api/users/{user_id}")
    assert response.status_code == 404  # yikes!
