from conftest import BASE_URL


class TestRecords:
    def test_list_records(self, api, auth_header):
        resp = api.get(f"{BASE_URL}/api/records", headers=auth_header)
        data = resp.json()
        assert data["code"] == 0

    def test_delete_record(self, api, auth_header):
        resp = api.delete(f"{BASE_URL}/api/records/99999", headers=auth_header)
        assert resp.json()["code"] == 40003
