import pytest
from conftest import BASE_URL


class TestTemplates:
    def test_list(self, api):
        resp = api.get(f"{BASE_URL}/api/templates")
        data = resp.json()
        assert data["code"] == 0
        assert "total" in data["data"]
        assert "items" in data["data"]
        assert "page" in data["data"]
        assert "page_size" in data["data"]

    def test_create(self, api, auth_header):
        resp = api.post(f"{BASE_URL}/api/templates", json={
            "name": "租赁合同",
        }, headers=auth_header)
        data = resp.json()
        assert data["code"] == 0
        assert data["data"]["id"] > 0

    def test_create_unauthorized(self, api):
        resp = api.post(f"{BASE_URL}/api/templates", json={"name": "test"})
        assert resp.json()["code"] == 40101

    def test_get(self, api, auth_header):
        create = api.post(f"{BASE_URL}/api/templates", json={"name": "test"}, headers=auth_header).json()
        tid = create["data"]["id"]
        resp = api.get(f"{BASE_URL}/api/templates/{tid}", headers=auth_header)
        data = resp.json()
        assert data["data"]["name"] == "test"

    def test_get_not_found(self, api, auth_header):
        resp = api.get(f"{BASE_URL}/api/templates/99999", headers=auth_header)
        assert resp.json()["code"] == 40003

    def test_update(self, api, auth_header):
        create = api.post(f"{BASE_URL}/api/templates", json={"name": "old"}, headers=auth_header).json()
        tid = create["data"]["id"]
        resp = api.put(f"{BASE_URL}/api/templates/{tid}", json={
            "name": "new name",
            "config_json": '{"controls":[],"rules":[]}',
        }, headers=auth_header)
        assert resp.json()["code"] == 0

    def test_delete(self, api, auth_header):
        create = api.post(f"{BASE_URL}/api/templates", json={"name": "del"}, headers=auth_header).json()
        tid = create["data"]["id"]
        resp = api.delete(f"{BASE_URL}/api/templates/{tid}", headers=auth_header)
        assert resp.json()["code"] == 0
        get = api.get(f"{BASE_URL}/api/templates/{tid}", headers=auth_header)
        assert get.json()["code"] == 40003

    def test_list_pagination(self, api, auth_header):
        for i in range(3):
            api.post(f"{BASE_URL}/api/templates", json={"name": f"t{i}"}, headers=auth_header)
        resp = api.get(f"{BASE_URL}/api/templates?page=1&page_size=2", headers=auth_header)
        data = resp.json()["data"]
        assert len(data["items"]) <= 2
        assert data["total"] >= 3
