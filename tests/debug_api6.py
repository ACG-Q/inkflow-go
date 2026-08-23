import requests
import json

BASE_URL = "http://localhost:8080"
s = requests.Session()

resp = s.post(f"{BASE_URL}/api/auth/login", json={"username": "admin", "password": "admin123"})
token = resp.json()["data"]["token"]
s.headers.update({"Authorization": f"Bearer {token}"})

# Create template exactly like the test
resp = s.post(f"{BASE_URL}/api/templates", json={"name": "侧边栏重构测试模板"})
data = resp.json()
tid = data["data"]["id"]
print(f"Created template: {tid}")

controls = [
    {
        "id": "test_textbox",
        "label": "测试文字",
        "type": "textbox",
        "x": 100.0,
        "y": 100.0,
        "width": 200.0,
        "height": 40.0,
        "font_size": 20,
        "font_family": "sans-serif",
        "required": False,
        "preview_text": "请输入",
        "check_size": 24,
    },
    {
        "id": "test_checkbox",
        "label": "测试选项",
        "type": "checkbox",
        "x": 100.0,
        "y": 200.0,
        "width": 24.0,
        "height": 24.0,
        "font_size": 14,
        "font_family": "sans-serif",
        "required": False,
        "check_size": 24,
    },
]

resp = s.put(f"{BASE_URL}/api/templates/{tid}", json={"controls": controls})
print(f"PUT status: {resp.status_code}")
print(f"PUT response: {resp.json()}")

# Verify
resp = s.get(f"{BASE_URL}/api/templates/{tid}")
data = resp.json()["data"]
print(f"Controls after: {len(data.get('controls', []))}")
for c in data.get("controls", []):
    print(f"  {c['id']}: {c['label']}")
