import os
import pytest
import requests

BASE_URL = os.environ.get("TEST_BASE_URL", "http://localhost:9876")


@pytest.fixture
def api():
    return requests.Session()


@pytest.fixture
def admin_token(api):
    resp = api.post(f"{BASE_URL}/api/auth/login", json={
        "username": "admin",
        "password": "admin123",
    })
    data = resp.json()
    assert data["code"] == 0
    return data["data"]["token"]


@pytest.fixture
def auth_header(admin_token):
    return {"Authorization": f"Bearer {admin_token}"}
