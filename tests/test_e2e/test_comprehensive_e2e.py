#!/usr/bin/env python3
"""
inkflow-go comprehensive E2E test.
Tests the full system: auth, templates, fonts, rules, signing, records, all pages.
Generates HTML report with screenshots.
"""

import os
import sys
import json
import time
import subprocess
import requests
from datetime import datetime
from pathlib import Path
from playwright.sync_api import sync_playwright

if sys.platform == "win32":
    import io
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', errors='replace')
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding='utf-8', errors='replace')

BASE_URL = os.environ.get("TEST_BASE_URL", "http://localhost:8080")
REPORT_DIR = Path(__file__).parent / "report"
SCREENSHOTS_DIR = REPORT_DIR / "screenshots"
PROJECT_ROOT = Path(__file__).parent.parent.parent
DATA_DIR = PROJECT_ROOT / "data"
PYTHON_REF_DIR = PROJECT_ROOT.parent / "contract-signature-system" / "static"


class E2ETest:
    def __init__(self):
        self.server_process = None
        self.session = requests.Session()
        self.admin_token = None
        self.screenshots = []
        self.test_results = []
        self.template_id = None
        self.bg_hash = "ee43bc355d5445698f97aceab727c814"

    def setup(self):
        print("=" * 60)
        print("inkflow-go E2E Test Setup")
        print("=" * 60)
        REPORT_DIR.mkdir(parents=True, exist_ok=True)
        SCREENSHOTS_DIR.mkdir(parents=True, exist_ok=True)
        self._start_server()
        self._wait_for_server()
        self._login_admin()
        self._upload_fonts()

    def teardown(self):
        if self.server_process:
            print("\nStopping server...")
            try:
                self.server_process.terminate()
                self.server_process.wait(timeout=5)
            except Exception:
                pass

    def _start_server(self):
        print("\n[1/4] Starting server...")
        try:
            resp = self.session.get(f"{BASE_URL}/api/health", timeout=2)
            if resp.status_code == 200:
                print("  Server already running!")
                return
        except requests.ConnectionError:
            pass

        exe = PROJECT_ROOT / "inkflow-go.exe"
        self.server_process = subprocess.Popen(
            [str(exe)],
            cwd=str(PROJECT_ROOT),
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            creationflags=subprocess.CREATE_NO_WINDOW if sys.platform == "win32" else 0,
        )
        print(f"  PID: {self.server_process.pid}")

    def _wait_for_server(self, timeout=30):
        print(f"\n[2/4] Waiting for server (timeout={timeout}s)...")
        start = time.time()
        while time.time() - start < timeout:
            try:
                resp = self.session.get(f"{BASE_URL}/api/health", timeout=2)
                if resp.status_code == 200:
                    print("  Ready!")
                    return
            except requests.ConnectionError:
                pass
            time.sleep(0.5)
        raise RuntimeError(f"Server did not start within {timeout}s")

    def _login_admin(self):
        print("\n[3/4] Logging in...")
        resp = self.session.post(f"{BASE_URL}/api/auth/login", json={"username": "admin", "password": "admin123"})
        data = resp.json()
        if data.get("code") == 0:
            self.admin_token = data["data"]["token"]
            self.session.headers.update({"Authorization": f"Bearer {self.admin_token}"})
            print("  Login OK")
        else:
            raise RuntimeError(f"Login failed: {data}")

    def _upload_fonts(self):
        print("\n[4/4] Uploading fonts...")
        fonts = [
            ("IndieFlower.ttf", "谷歌手写体"),
            ("PingFangFangMaoTiCaoShu-2.ttf", "平方仿毛体草书"),
        ]
        for fname, display in fonts:
            fpath = PYTHON_REF_DIR / "fonts" / fname
            if not fpath.exists():
                print(f"  Font not found: {fpath}")
                continue
            with open(fpath, "rb") as f:
                resp = self.session.post(
                    f"{BASE_URL}/api/fonts",
                    files={"file": (fname, f, "font/ttf")},
                    data={"display_name": display},
                )
            r = resp.json()
            if r.get("code") == 0:
                print(f"  Uploaded: {display}")
            elif "已存在" in r.get("message", ""):
                print(f"  Already exists: {display}")
            else:
                print(f"  Upload failed: {r}")

    def _screenshot(self, page, name, label):
        path = SCREENSHOTS_DIR / f"{name}.png"
        page.screenshot(path=str(path), full_page=True)
        self.screenshots.append((label, f"{name}.png"))
        print(f"  Screenshot: {name}.png")

    def _api(self, method, path, **kwargs):
        resp = self.session.request(method, f"{BASE_URL}{path}", **kwargs)
        return resp.json()

    def run(self):
        print("\n" + "=" * 60)
        print("Running E2E Tests")
        print("=" * 60)

        tests = [
            ("01_auth_login", self.test_01_auth_login),
            ("02_template_create", self.test_02_template_create),
            ("03_template_upload_bg", self.test_03_template_upload_bg),
            ("04_template_get", self.test_04_template_get),
            ("05_template_update_name", self.test_05_template_update_name),
            ("06_controls_add", self.test_06_controls_add),
            ("07_rules_add", self.test_07_rules_add),
            ("08_handwriting_global", self.test_08_handwriting_global),
            ("09_handwriting_template", self.test_09_handwriting_template),
            ("10_sign_with_override", self.test_10_sign_with_override),
            ("11_records_list", self.test_11_records_list),
            ("12_template_delete", self.test_12_template_delete),
            ("13_fonts_list", self.test_13_fonts_list),
            ("14_stats", self.test_14_stats),
            ("15_page_home", self.test_15_page_home),
            ("16_page_admin_dashboard", self.test_16_page_admin_dashboard),
            ("17_page_admin_editor", self.test_17_page_admin_editor),
            ("18_page_admin_settings", self.test_18_page_admin_settings),
            ("19_page_admin_fonts", self.test_19_page_admin_fonts),
            ("20_page_admin_records", self.test_20_page_admin_records),
            ("21_page_sign_front", self.test_21_page_sign_front),
            ("22_sign_modal_open", self.test_22_sign_modal_open),
        ]

        for name, func in tests:
            print(f"\n--- {name} ---")
            try:
                func()
                self.test_results.append({"name": name, "status": "PASS", "error": None})
            except Exception as e:
                self.test_results.append({"name": name, "status": "FAIL", "error": str(e)})
                print(f"  FAIL: {e}")

    # --- Auth ---
    def test_01_auth_login(self):
        r = self._api("POST", "/api/auth/login", json={"username": "admin", "password": "admin123"})
        assert r["code"] == 0
        r2 = self._api("GET", "/api/auth/me")
        assert r2["code"] == 0
        print(f"  User: {r2['data']['username']}")

    # --- Template CRUD ---
    def test_02_template_create(self):
        r = self._api("POST", "/api/templates", json={"name": "E2E测试-租赁合同"})
        assert r["code"] == 0
        self.template_id = r["data"]["id"]
        print(f"  Created template #{self.template_id}")

    def test_03_template_upload_bg(self):
        bg_path = DATA_DIR / "bg_images" / self.bg_hash / "page_1.png"
        assert bg_path.exists(), f"BG not found: {bg_path}"
        with open(bg_path, "rb") as f:
            resp = self.session.post(
                f"{BASE_URL}/api/templates/{self.template_id}/upload",
                files={"file": ("page_1.png", f, "image/png")},
            )
        r = resp.json()
        assert r["code"] == 0
        print(f"  BG uploaded: {r['data']['bg_image']}")

    def test_04_template_get(self):
        r = self._api("GET", f"/api/templates/{self.template_id}")
        assert r["code"] == 0
        t = r["data"]
        assert t["name"] == "E2E测试-租赁合同"
        assert t["bg_image"] != ""
        assert "handwriting" in t
        print(f"  Name={t['name']}, bg={t['bg_image'][:40]}..., hw_keys={list(t['handwriting'].keys())[:5]}")

    def test_05_template_update_name(self):
        r = self._api("PUT", f"/api/templates/{self.template_id}", json={"name": "E2E测试-房屋租赁合同"})
        assert r["code"] == 0
        r2 = self._api("GET", f"/api/templates/{self.template_id}")
        assert r2["data"]["name"] == "E2E测试-房屋租赁合同"
        print(f"  Renamed to: {r2['data']['name']}")

    # --- Controls ---
    def test_06_controls_add(self):
        controls = [
            {"id": "ctrl_name", "label": "甲方名称", "type": "textbox", "x": 308, "y": 201, "width": 335, "height": 40, "font_size": 35, "font_family": "sans-serif", "required": True, "preview_text": "", "check_size": 24},
            {"id": "ctrl_addr", "label": "租赁地址", "type": "textbox", "x": 305, "y": 339, "width": 604, "height": 34, "font_size": 30, "font_family": "sans-serif", "required": True, "preview_text": "", "check_size": 24},
            {"id": "ctrl_chk1", "label": "选项一", "type": "checkbox", "x": 228, "y": 418, "width": 35, "height": 35, "font_size": 20, "font_family": "sans-serif", "required": False, "preview_text": "", "check_size": 24},
            {"id": "ctrl_chk2", "label": "选项二", "type": "checkbox", "x": 328, "y": 418, "width": 35, "height": 35, "font_size": 20, "font_family": "sans-serif", "required": False, "preview_text": "", "check_size": 24},
        ]
        r = self._api("PUT", f"/api/templates/{self.template_id}", json={"controls": controls})
        assert r["code"] == 0
        r2 = self._api("GET", f"/api/templates/{self.template_id}")
        assert len(r2["data"]["controls"]) == 4
        print(f"  Added {len(r2['data']['controls'])} controls")

    # --- Rules ---
    def test_07_rules_add(self):
        rules = [
            {"id": "rule_req_name", "type": "required", "name": "甲方名称必填", "target": "ctrl_name", "config": {}},
            {"id": "rule_req_addr", "type": "required", "name": "地址必填", "target": "ctrl_addr", "config": {}},
            {"id": "rule_mutual", "type": "mutual_exclusion", "name": "选项互斥", "target": "", "config": {"targets": ["ctrl_chk1", "ctrl_chk2"]}},
            {"id": "rule_date", "type": "auto_fill", "name": "自动填充日期", "target": "ctrl_name", "config": {"source": "system.date", "format": "YYYY-MM-DD"}},
        ]
        r = self._api("PUT", f"/api/templates/{self.template_id}", json={"rules": rules})
        assert r["code"] == 0
        r2 = self._api("GET", f"/api/templates/{self.template_id}")
        assert len(r2["data"]["rules"]) == 4
        print(f"  Added {len(r2['data']['rules'])} rules")

    # --- Handwriting Global ---
    def test_08_handwriting_global(self):
        hw = {
            "font_family": "谷歌手写体",
            "paper_enabled": True, "paper_opacity": 0.20, "fiber_count": 250, "dot_count": 900,
            "global_tilt": 1.5, "baseline_drift": 1.0, "char_jitter": 2.5, "char_rotation": 2.0,
            "ink_opacity_min": 0.80, "ink_opacity_max": 0.95, "char_spacing": 1.8,
            "ink_spots_enabled": True, "ink_spots_chance": 0.18, "ink_spots_max": 3,
            "shadow_blur": 0.9, "checkbox_enabled": True,
        }
        r = self._api("PUT", "/api/settings/handwriting", json=hw)
        assert r["code"] == 0
        r2 = self._api("GET", "/api/settings/handwriting")
        assert r2["data"]["font_family"] == "谷歌手写体"
        assert r2["data"]["paper_opacity"] == 0.20
        print(f"  Global: font={r2['data']['font_family']}, opacity={r2['data']['paper_opacity']}")

    # --- Handwriting Template ---
    def test_09_handwriting_template(self):
        hw = {"font_family": "平方仿毛体草书", "paper_enabled": False, "char_jitter": 1.0, "ink_spots_enabled": False}
        r = self._api("PUT", f"/api/templates/{self.template_id}", json={"handwriting": hw})
        assert r["code"] == 0
        r2 = self._api("GET", f"/api/templates/{self.template_id}")
        assert r2["data"]["handwriting"]["font_family"] == "平方仿毛体草书"
        assert r2["data"]["handwriting"]["paper_enabled"] is False
        print(f"  Template: font={r2['data']['handwriting']['font_family']}, paper={r2['data']['handwriting']['paper_enabled']}")

    # --- Sign with override ---
    def test_10_sign_with_override(self):
        override = {"paper_opacity": 0.35, "char_jitter": 4.0}
        r = self._api("POST", f"/api/templates/{self.template_id}/sign", json={
            "fields_data": {"ctrl_name": "张三", "ctrl_addr": "北京市朝阳区", "ctrl_chk1": True},
            "effect_preset": "light",
            "text_layer_data": "",
            "handwriting": override,
        })
        assert r["code"] == 0
        assert "image_url" in r["data"]
        print(f"  Signed: {r['data']['image_url']}")

    # --- Records ---
    def test_11_records_list(self):
        r = self._api("GET", "/api/records")
        assert r["code"] == 0
        items = r["data"]["items"]
        assert len(items) >= 1
        print(f"  Records: {r['data']['total']} total")

    # --- Delete template ---
    def test_12_template_delete(self):
        r_create = self._api("POST", "/api/templates", json={"name": "待删除模板"})
        tid = r_create["data"]["id"]
        r = self._api("DELETE", f"/api/templates/{tid}")
        assert r["code"] == 0
        print(f"  Deleted template #{tid}")

    # --- Fonts ---
    def test_13_fonts_list(self):
        r = self._api("GET", "/api/fonts")
        assert r["code"] == 0
        print(f"  Fonts: {len(r['data'])} total")

    # --- Stats ---
    def test_14_stats(self):
        r = self._api("GET", "/api/admin/stats")
        assert r["code"] == 0
        s = r["data"]
        assert s["template_count"] >= 1
        print(f"  Stats: templates={s['template_count']}, records={s['record_count']}, fonts={s['font_count']}")

    # --- Pages ---
    def test_15_page_home(self):
        with sync_playwright() as p:
            browser = p.chromium.launch(channel="chrome", headless=True)
            page = browser.new_page(viewport={"width": 1280, "height": 800})
            page.goto(f"{BASE_URL}/", wait_until="networkidle")
            page.wait_for_timeout(2000)
            self._screenshot(page, "15_home", "首页")
            browser.close()

    def test_16_page_admin_dashboard(self):
        with sync_playwright() as p:
            browser = p.chromium.launch(channel="chrome", headless=True)
            page = browser.new_page(viewport={"width": 1280, "height": 800})
            page.goto(f"{BASE_URL}/admin/login", wait_until="networkidle")
            page.wait_for_selector("input[type='text']", timeout=10000)
            page.fill("input[type='text']", "admin")
            page.fill("input[type='password']", "admin123")
            page.click("button[type='submit']")
            page.wait_for_url(f"{BASE_URL}/admin", timeout=10000)
            page.wait_for_timeout(2000)
            self._screenshot(page, "16_admin_dashboard", "管理后台-仪表盘")
            browser.close()

    def test_17_page_admin_editor(self):
        if not self.template_id:
            return
        with sync_playwright() as p:
            browser = p.chromium.launch(channel="chrome", headless=True)
            page = browser.new_page(viewport={"width": 1280, "height": 800})
            page.goto(f"{BASE_URL}/admin/login", wait_until="networkidle")
            page.fill("input[type='text']", "admin")
            page.fill("input[type='password']", "admin123")
            page.click("button[type='submit']")
            page.wait_for_url(f"{BASE_URL}/admin", timeout=10000)
            page.goto(f"{BASE_URL}/admin/editor?id={self.template_id}", wait_until="networkidle")
            page.wait_for_timeout(3000)
            self._screenshot(page, "17_admin_editor", "管理后台-编辑器")

            hw_btn = page.locator("text=手写效果")
            if hw_btn.is_visible():
                hw_btn.click()
                page.wait_for_timeout(1000)
                self._screenshot(page, "17b_editor_hw_modal", "编辑器-手写效果弹窗")
            browser.close()

    def test_18_page_admin_settings(self):
        with sync_playwright() as p:
            browser = p.chromium.launch(channel="chrome", headless=True)
            page = browser.new_page(viewport={"width": 1280, "height": 800})
            page.goto(f"{BASE_URL}/admin/login", wait_until="networkidle")
            page.fill("input[type='text']", "admin")
            page.fill("input[type='password']", "admin123")
            page.click("button[type='submit']")
            page.wait_for_url(f"{BASE_URL}/admin", timeout=10000)
            page.goto(f"{BASE_URL}/admin/settings", wait_until="networkidle")
            page.wait_for_timeout(2000)
            self._screenshot(page, "18_admin_settings", "管理后台-全局设置")
            browser.close()

    def test_19_page_admin_fonts(self):
        with sync_playwright() as p:
            browser = p.chromium.launch(channel="chrome", headless=True)
            page = browser.new_page(viewport={"width": 1280, "height": 800})
            page.goto(f"{BASE_URL}/admin/login", wait_until="networkidle")
            page.fill("input[type='text']", "admin")
            page.fill("input[type='password']", "admin123")
            page.click("button[type='submit']")
            page.wait_for_url(f"{BASE_URL}/admin", timeout=10000)
            page.goto(f"{BASE_URL}/admin/fonts", wait_until="networkidle")
            page.wait_for_timeout(2000)
            self._screenshot(page, "19_admin_fonts", "管理后台-字体管理")
            browser.close()

    def test_20_page_admin_records(self):
        with sync_playwright() as p:
            browser = p.chromium.launch(channel="chrome", headless=True)
            page = browser.new_page(viewport={"width": 1280, "height": 800})
            page.goto(f"{BASE_URL}/admin/login", wait_until="networkidle")
            page.fill("input[type='text']", "admin")
            page.fill("input[type='password']", "admin123")
            page.click("button[type='submit']")
            page.wait_for_url(f"{BASE_URL}/admin", timeout=10000)
            page.goto(f"{BASE_URL}/admin/records", wait_until="networkidle")
            page.wait_for_timeout(2000)
            self._screenshot(page, "20_admin_records", "管理后台-签署记录")
            browser.close()

    def test_21_page_sign_front(self):
        if not self.template_id:
            return
        with sync_playwright() as p:
            browser = p.chromium.launch(channel="chrome", headless=True)
            page = browser.new_page(viewport={"width": 1280, "height": 800})
            page.goto(f"{BASE_URL}/sign?template_id={self.template_id}", wait_until="networkidle")
            page.wait_for_timeout(3000)
            self._screenshot(page, "21_sign_front", "签名页")
            browser.close()

    def test_22_sign_modal_open(self):
        if not self.template_id:
            return
        with sync_playwright() as p:
            browser = p.chromium.launch(channel="chrome", headless=True)
            page = browser.new_page(viewport={"width": 1280, "height": 800})
            page.goto(f"{BASE_URL}/sign?template_id={self.template_id}", wait_until="networkidle")
            page.wait_for_timeout(2000)
            btn = page.locator("text=调整手写效果")
            if btn.is_visible():
                btn.click()
                page.wait_for_timeout(1000)
                self._screenshot(page, "22_sign_hw_modal", "签名页-手写效果弹窗")
            browser.close()

    def generate_report(self):
        print("\n" + "=" * 60)
        print("Generating Report")
        print("=" * 60)

        passed = sum(1 for r in self.test_results if r["status"] == "PASS")
        failed = sum(1 for r in self.test_results if r["status"] == "FAIL")
        total = len(self.test_results)

        rows = ""
        for r in self.test_results:
            cls = "pass" if r["status"] == "PASS" else "fail"
            icon = "PASS" if r["status"] == "PASS" else "FAIL"
            err = f'<div class="error">{r["error"]}</div>' if r["error"] else ""
            rows += f'<div class="test-item"><div class="status {cls}">{icon}</div><div class="name">{r["name"]}{err}</div></div>\n'

        imgs = ""
        for title, fname in self.screenshots:
            imgs += f'<div class="screenshot-card"><img src="screenshots/{fname}" alt="{title}" loading="lazy"><div class="caption">{title}</div></div>\n'

        html = f"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>inkflow-go E2E 测试报告</title>
<style>
*{{margin:0;padding:0;box-sizing:border-box}}
body{{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#f5f5f5;color:#333;line-height:1.6}}
.container{{max-width:1200px;margin:0 auto;padding:20px}}
header{{background:linear-gradient(135deg,#667eea 0%,#764ba2 100%);color:#fff;padding:30px;border-radius:10px;margin-bottom:30px}}
header h1{{font-size:24px;margin-bottom:10px}}
header .meta{{opacity:.9;font-size:14px}}
.summary{{display:grid;grid-template-columns:repeat(3,1fr);gap:20px;margin-bottom:30px}}
.summary-card{{background:#fff;padding:20px;border-radius:8px;box-shadow:0 2px 4px rgba(0,0,0,.1);text-align:center}}
.summary-card .number{{font-size:36px;font-weight:700}}
.summary-card .label{{color:#666;font-size:14px}}
.summary-card.pass .number{{color:#22c55e}}
.summary-card.fail .number{{color:#ef4444}}
.summary-card.total .number{{color:#3b82f6}}
.test-list{{background:#fff;border-radius:8px;box-shadow:0 2px 4px rgba(0,0,0,.1);overflow:hidden;margin-bottom:30px}}
.test-list h2{{padding:15px 20px;border-bottom:1px solid #eee;font-size:16px}}
.test-item{{padding:12px 20px;border-bottom:1px solid #eee;display:flex;align-items:center}}
.test-item:last-child{{border-bottom:none}}
.test-item .status{{width:52px;height:24px;border-radius:4px;margin-right:15px;display:flex;align-items:center;justify-content:center;font-size:11px;font-weight:600;color:#fff;flex-shrink:0}}
.test-item .status.pass{{background:#22c55e}}
.test-item .status.fail{{background:#ef4444}}
.test-item .name{{flex:1;font-size:14px}}
.test-item .error{{color:#ef4444;font-size:12px;margin-top:4px}}
.screenshots{{margin-bottom:30px}}
.screenshots h2{{margin-bottom:20px;font-size:16px}}
.screenshot-grid{{display:grid;grid-template-columns:repeat(auto-fill,minmax(400px,1fr));gap:20px}}
.screenshot-card{{background:#fff;border-radius:8px;box-shadow:0 2px 4px rgba(0,0,0,.1);overflow:hidden}}
.screenshot-card img{{width:100%;height:auto;display:block}}
.screenshot-card .caption{{padding:10px 15px;background:#f8f9fa;font-size:13px;color:#555}}
footer{{text-align:center;padding:20px;color:#888;font-size:13px}}
</style>
</head>
<body>
<div class="container">
<header>
<h1>inkflow-go E2E 测试报告</h1>
<div class="meta">
<div>生成时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</div>
<div>测试环境: {BASE_URL}</div>
</div>
</header>
<div class="summary">
<div class="summary-card total"><div class="number">{total}</div><div class="label">总测试数</div></div>
<div class="summary-card pass"><div class="number">{passed}</div><div class="label">通过</div></div>
<div class="summary-card fail"><div class="number">{failed}</div><div class="label">失败</div></div>
</div>
<div class="test-list">
<h2>测试详情</h2>
{rows}
</div>
<div class="screenshots">
<h2>截图记录</h2>
<div class="screenshot-grid">
{imgs}
</div>
</div>
<footer>inkflow-go E2E Test Report</footer>
</div>
</body>
</html>"""

        path = REPORT_DIR / "test_report.html"
        with open(path, "w", encoding="utf-8") as f:
            f.write(html)
        print(f"  Report: {path}")
        return path


def main():
    test = E2ETest()
    try:
        test.setup()
        test.run()
    except Exception as e:
        print(f"\nFatal: {e}")
        import traceback
        traceback.print_exc()
    finally:
        test.teardown()
        test.generate_report()


if __name__ == "__main__":
    main()
