import pytest
from conftest import BASE_URL


class TestAuth:
    def test_login_success(self, api):
        resp = api.post(f"{BASE_URL}/api/auth/login", json={
            "username": "admin",
            "password": "admin123",
        })
        data = resp.json()
        assert data["code"] == 0
        assert "token" in data["data"]

    def test_login_wrong_password(self, api):
        resp = api.post(f"{BASE_URL}/api/auth/login", json={
            "username": "admin",
            "password": "wrong",
        })
        assert resp.json()["code"] == 40102

    def test_login_missing_fields(self, api):
        resp = api.post(f"{BASE_URL}/api/auth/login", json={})
        assert resp.json()["code"] == 40001

    def test_me_unauthorized(self, api):
        resp = api.get(f"{BASE_URL}/api/auth/me")
        assert resp.json()["code"] == 40101

    def test_me_authorized(self, api, auth_header):
        resp = api.get(f"{BASE_URL}/api/auth/me", headers=auth_header)
        assert resp.json()["code"] == 0
