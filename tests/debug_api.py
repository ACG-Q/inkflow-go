import requests
import json

BASE_URL = "http://localhost:8080"
s = requests.Session()

# Login
resp = s.post(f"{BASE_URL}/api/auth/login", json={"username": "admin", "password": "admin123"})
token = resp.json()["data"]["token"]
s.headers.update({"Authorization": f"Bearer {token}"})

# Create template
resp = s.post(f"{BASE_URL}/api/templates", json={"name": "API测试模板"})
data = resp.json()
print(f"Create template: {data}")
tid = data["data"]["id"]

# Add controls
controls = [
    {
        "id": "textbox_1",
        "label": "甲方名称",
        "type": "textbox",
        "x": 100,
        "y": 100,
        "width": 200,
        "height": 40,
        "font_size": 20,
        "font_family": "sans-serif",
        "required": False,
        "preview_text": "请输入",
        "check_size": 24,
    },
]

resp = s.put(f"{BASE_URL}/api/templates/{tid}", json={"controls": controls})
print(f"Update response: {resp.json()}")

# Verify
resp = s.get(f"{BASE_URL}/api/templates/{tid}")
data = resp.json()["data"]
print(f"Controls after update: {len(data.get('controls', []))}")
for c in data.get("controls", []):
    print(f"  - {c['id']}: {c['label']}")
