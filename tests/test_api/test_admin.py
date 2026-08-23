from conftest import BASE_URL


class TestAdmin:
    def test_stats(self, api, auth_header):
        resp = api.get(f"{BASE_URL}/api/admin/stats", headers=auth_header)
        data = resp.json()
        assert data["code"] == 0
        assert "template_count" in data["data"]
        assert "record_count" in data["data"]
        assert "font_count" in data["data"]
