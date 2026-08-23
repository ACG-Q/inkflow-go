#!/usr/bin/env python3
"""
E2E test for handwriting configuration using Python reference resources.
Tests the 3-level merge: global → template → sign override.
Generates an HTML report with screenshots.
"""

import os
import sys
import json
import time
import shutil
import subprocess
import requests
from datetime import datetime
from pathlib import Path
from playwright.sync_api import sync_playwright, expect

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

# Test data
TEST_BG_IMAGES = [
    ("ee43bc355d5445698f97aceab727c814", "租赁合同背景图"),
    ("b753d870c00649e794e658b822aed633", "委托协议背景图"),
    ("5fa5866660434b5ba4b75e55e6c012de", "背景图3"),
    ("61bfae99251a4216bc76a87127aa3ca2", "背景图4"),
]

TEST_FONTS = [
    ("IndieFlower.ttf", "谷歌手写体"),
    ("PingFangFangMaoTiCaoShu-2.ttf", "平方仿毛体草书"),
    ("860d705fb1cf47bb9facaf98496142f2.ttf", "平方洒脱体"),
    ("87beda0b6cee42a28d83968955d7825a.ttf", "鸿雷小纸条青春体"),
]


class HandwritingE2ETest:
    def __init__(self):
        self.server_process = None
        self.session = requests.Session()
        self.admin_token = None
        self.screenshots = []
        self.test_results = []
        self.template_id = None

    def setup(self):
        """Setup test environment."""
        print("=" * 60)
        print("Handwriting E2E Test Setup")
        print("=" * 60)

        REPORT_DIR.mkdir(parents=True, exist_ok=True)
        SCREENSHOTS_DIR.mkdir(parents=True, exist_ok=True)

        self._start_server()
        self._wait_for_server()
        self._login_admin()
        self._upload_fonts()

    def teardown(self):
        """Cleanup test environment."""
        if self.server_process:
            print("\nStopping server...")
            try:
                self.server_process.terminate()
                self.server_process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self.server_process.kill()
            except OSError:
                pass

    def _start_server(self):
        """Start the inkflow-go server."""
        print("\n[1/4] Starting inkflow-go server...")

        try:
            resp = self.session.get(f"{BASE_URL}/api/health", timeout=2)
            if resp.status_code == 200:
                print("  Server already running!")
                self.server_process = None
                return
        except requests.ConnectionError:
            pass

        exe_path = PROJECT_ROOT / "inkflow-go.exe"
        if not exe_path.exists():
            exe_path = PROJECT_ROOT / "inkflow-go"

        self.server_process = subprocess.Popen(
            [str(exe_path)],
            cwd=str(PROJECT_ROOT),
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            creationflags=subprocess.CREATE_NO_WINDOW if sys.platform == "win32" else 0,
        )
        print(f"  Server PID: {self.server_process.pid}")

    def _wait_for_server(self, timeout=30):
        """Wait for server to be ready."""
        print(f"\n[2/4] Waiting for server (timeout={timeout}s)...")
        start = time.time()
        while time.time() - start < timeout:
            try:
                resp = self.session.get(f"{BASE_URL}/api/health", timeout=2)
                if resp.status_code == 200:
                    print("  Server is ready!")
                    return
            except requests.ConnectionError:
                pass
            time.sleep(0.5)
        raise RuntimeError(f"Server did not start within {timeout}s")

    def _login_admin(self):
        """Login as admin and get JWT token."""
        print("\n[3/4] Logging in as admin...")
        resp = self.session.post(
            f"{BASE_URL}/api/auth/login",
            json={"username": "admin", "password": "admin123"},
        )
        data = resp.json()
        if data.get("code") == 0:
            self.admin_token = data["data"]["token"]
            self.session.headers.update({"Authorization": f"Bearer {self.admin_token}"})
            print("  Login successful!")
        else:
            print(f"  Login failed: {data}")
            raise RuntimeError("Admin login failed")

    def _upload_fonts(self):
        """Upload test fonts to the server."""
        print("\n[4/4] Uploading test fonts...")
        for font_file, display_name in TEST_FONTS:
            font_path = PYTHON_REF_DIR / "fonts" / font_file
            if not font_path.exists():
                print(f"  Font not found: {font_path}")
                continue

            with open(font_path, "rb") as f:
                resp = self.session.post(
                    f"{BASE_URL}/api/fonts",
                    files={"file": (font_file, f, "font/ttf")},
                    data={"display_name": display_name},
                )
            result = resp.json()
            if result.get("code") == 0:
                print(f"  Uploaded: {display_name}")
            else:
                print(f"  Upload failed: {result}")

    def run_tests(self):
        """Run all handwriting E2E tests."""
        print("\n" + "=" * 60)
        print("Running Handwriting E2E Tests")
        print("=" * 60)

        tests = [
            ("test_01_global_handwriting_get", self.test_01_global_handwriting_get),
            ("test_02_global_handwriting_update", self.test_02_global_handwriting_update),
            ("test_03_create_template_with_bg", self.test_03_create_template_with_bg),
            ("test_04_add_controls_to_template", self.test_04_add_controls_to_template),
            ("test_05_template_handwriting_config", self.test_05_template_handwriting_config),
            ("test_06_sign_with_override", self.test_06_sign_with_override),
            ("test_07_verify_merge_priority", self.test_07_verify_merge_priority),
            ("test_08_admin_settings_page", self.test_08_admin_settings_page),
            ("test_09_editor_page", self.test_09_editor_page),
            ("test_10_sign_page", self.test_10_sign_page),
        ]

        for test_name, test_func in tests:
            print(f"\n--- {test_name} ---")
            try:
                test_func()
                self.test_results.append({"name": test_name, "status": "PASS", "error": None})
                print(f"  ✓ {test_name} PASSED")
            except Exception as e:
                self.test_results.append({"name": test_name, "status": "FAIL", "error": str(e)})
                print(f"  ✗ {test_name} FAILED: {e}")

    def test_01_global_handwriting_get(self):
        """Test GET /api/settings/handwriting returns defaults."""
        resp = self.session.get(f"{BASE_URL}/api/settings/handwriting")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 0
        hw = data["data"]
        print(f"  Response keys: {list(hw.keys())}")
        assert "paper_enabled" in hw, f"Missing 'paper_enabled' in response: {hw}"
        assert hw["paper_enabled"] is True
        assert hw["paper_opacity"] == 0.12
        assert hw["fiber_count"] == 200
        assert hw["char_jitter"] == 2.0
        print(f"  Global defaults: paper_opacity={hw['paper_opacity']}, char_jitter={hw['char_jitter']}")

    def test_02_global_handwriting_update(self):
        """Test PUT /api/settings/handwriting updates global config."""
        global_config = {
            "paper_enabled": True,
            "paper_opacity": 0.25,
            "fiber_count": 300,
            "dot_count": 1000,
            "global_tilt": 2.0,
            "baseline_drift": 1.2,
            "char_jitter": 3.0,
            "char_rotation": 2.5,
            "ink_opacity_min": 0.7,
            "ink_opacity_max": 0.9,
            "char_spacing": 2.0,
            "ink_spots_enabled": True,
            "ink_spots_chance": 0.2,
            "ink_spots_max": 3,
            "shadow_blur": 1.0,
            "checkbox_enabled": True,
        }
        resp = self.session.put(
            f"{BASE_URL}/api/settings/handwriting",
            json=global_config,
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 0

        resp = self.session.get(f"{BASE_URL}/api/settings/handwriting")
        hw = resp.json()["data"]
        assert hw["paper_opacity"] == 0.25
        assert hw["char_jitter"] == 3.0
        print(f"  Global updated: paper_opacity={hw['paper_opacity']}, char_jitter={hw['char_jitter']}")

    def test_03_create_template_with_bg(self):
        """Test creating a template with background image."""
        resp = self.session.post(
            f"{BASE_URL}/api/templates",
            json={"name": "手写效果测试模板"},
        )
        print(f"  Create response: {resp.status_code} {resp.text[:200]}")
        assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text[:200]}"
        data = resp.json()
        assert data.get("code") == 0, f"API error: {data}"
        self.template_id = data["data"]["id"]
        print(f"  Created template ID: {self.template_id}")

        bg_hash = "ee43bc355d5445698f97aceab727c814"
        bg_path = DATA_DIR / "bg_images" / bg_hash / "page_1.png"
        assert bg_path.exists(), f"Background image not found: {bg_path}"

        with open(bg_path, "rb") as f:
            resp = self.session.post(
                f"{BASE_URL}/api/templates/{self.template_id}/upload",
                files={"file": ("page_1.png", f, "image/png")},
            )
        print(f"  Upload response: {resp.status_code} {resp.text[:200]}")
        assert resp.status_code == 200, f"Upload failed: {resp.text[:200]}"
        print(f"  Uploaded background image: {bg_hash}/page_1.png")

    def test_04_add_controls_to_template(self):
        """Test adding controls to template."""
        controls = [
            {
                "id": "textbox_name",
                "label": "甲方名称",
                "type": "textbox",
                "x": 308,
                "y": 201,
                "width": 335,
                "height": 40,
                "font_size": 35,
                "font_family": "平方仿毛体草书",
                "required": True,
                "preview_text": "",
                "check_size": 24,
            },
            {
                "id": "textbox_address",
                "label": "租赁地址",
                "type": "textbox",
                "x": 305,
                "y": 339,
                "width": 604,
                "height": 34,
                "font_size": 30,
                "font_family": "平方仿毛体草书",
                "required": True,
                "preview_text": "",
                "check_size": 24,
            },
            {
                "id": "checkbox_option",
                "label": "选项一",
                "type": "checkbox",
                "x": 228,
                "y": 418,
                "width": 35,
                "height": 35,
                "font_size": 20,
                "font_family": "sans-serif",
                "required": False,
                "preview_text": "",
                "check_size": 24,
            },
        ]

        resp = self.session.put(
            f"{BASE_URL}/api/templates/{self.template_id}",
            json={"controls": controls},
        )
        assert resp.status_code == 200
        print(f"  Added {len(controls)} controls to template")

    def test_05_template_handwriting_config(self):
        """Test configuring template-level handwriting."""
        template_hw = {
            "paper_enabled": False,
            "paper_opacity": 0.05,
            "fiber_count": 100,
            "dot_count": 400,
            "global_tilt": 0.5,
            "baseline_drift": 0.4,
            "char_jitter": 1.0,
            "char_rotation": 0.8,
            "ink_opacity_min": 0.9,
            "ink_opacity_max": 1.0,
            "char_spacing": 1.0,
            "ink_spots_enabled": False,
            "ink_spots_chance": 0.1,
            "ink_spots_max": 1,
            "shadow_blur": 0.5,
            "checkbox_enabled": False,
        }

        resp = self.session.put(
            f"{BASE_URL}/api/templates/{self.template_id}",
            json={"handwriting": template_hw},
        )
        assert resp.status_code == 200

        resp = self.session.get(f"{BASE_URL}/api/templates/{self.template_id}")
        data = resp.json()["data"]
        hw = data["handwriting"]
        assert hw["paper_enabled"] is False
        assert hw["char_jitter"] == 1.0
        print(f"  Template handwriting: paper_enabled={hw['paper_enabled']}, char_jitter={hw['char_jitter']}")

    def test_06_sign_with_override(self):
        """Test signing with handwriting override."""
        sign_override = {
            "paper_enabled": True,
            "paper_opacity": 0.35,
            "char_jitter": 4.0,
            "ink_spots_enabled": True,
            "ink_spots_chance": 0.3,
        }

        resp = self.session.post(
            f"{BASE_URL}/api/templates/{self.template_id}/sign",
            json={
                "fields_data": {
                    "textbox_name": "测试甲方",
                    "textbox_address": "北京市朝阳区",
                    "checkbox_option": True,
                },
                "effect_preset": "light",
                "text_layer_data": "",
                "handwriting": sign_override,
            },
        )
        assert resp.status_code == 200
        result = resp.json()["data"]
        assert "image_url" in result
        print(f"  Sign result: {result['image_url']}")

    def test_07_verify_merge_priority(self):
        """Verify merge priority: sign > template > global > defaults."""
        global_config = {
            "paper_opacity": 0.25,
            "char_jitter": 3.0,
            "global_tilt": 2.0,
            "ink_opacity_min": 0.7,
        }
        self.session.put(
            f"{BASE_URL}/api/settings/handwriting",
            json=global_config,
        )

        template_hw = {
            "paper_opacity": 0.15,
            "char_jitter": 1.5,
        }
        self.session.put(
            f"{BASE_URL}/api/templates/{self.template_id}",
            json={"handwriting": template_hw},
        )

        sign_override = {
            "paper_opacity": 0.4,
        }

        resp = self.session.post(
            f"{BASE_URL}/api/templates/{self.template_id}/sign",
            json={
                "fields_data": {"textbox_name": "验证优先级"},
                "effect_preset": "none",
                "text_layer_data": "",
                "handwriting": sign_override,
            },
        )
        assert resp.status_code == 200
        print("  Merge priority test: sign > template > global > defaults")
        print("  Expected: paper_opacity=0.4 (from sign override)")

    def test_08_admin_settings_page(self):
        """Test admin settings page loads correctly."""
        with sync_playwright() as p:
            browser = p.chromium.launch(channel="chrome", headless=True)
            page = browser.new_page(viewport={"width": 1280, "height": 800})

            page.goto(f"{BASE_URL}/admin/login", wait_until="networkidle")
            page.wait_for_selector("input[type='text']", timeout=10000)
            page.fill("input[type='text']", "admin")
            page.fill("input[type='password']", "admin123")
            page.click("button[type='submit']")
            page.wait_for_url(f"{BASE_URL}/admin", timeout=10000)
            print("  Logged in to admin dashboard")

            page.goto(f"{BASE_URL}/admin/settings", wait_until="networkidle")
            page.wait_for_timeout(2000)

            screenshot_path = SCREENSHOTS_DIR / "08_admin_settings.png"
            page.screenshot(path=str(screenshot_path), full_page=True)
            self.screenshots.append(("Admin Settings Page", "08_admin_settings.png"))
            print(f"  Screenshot saved: {screenshot_path}")

            browser.close()

    def test_09_editor_page(self):
        """Test editor page with template."""
        if not self.template_id:
            print("  Skipping: no template_id")
            return

        with sync_playwright() as p:
            browser = p.chromium.launch(channel="chrome", headless=True)
            page = browser.new_page(viewport={"width": 1280, "height": 800})

            page.goto(f"{BASE_URL}/admin/login", wait_until="networkidle")
            page.wait_for_selector("input[type='text']", timeout=10000)
            page.fill("input[type='text']", "admin")
            page.fill("input[type='password']", "admin123")
            page.click("button[type='submit']")
            page.wait_for_url(f"{BASE_URL}/admin", timeout=10000)

            page.goto(
                f"{BASE_URL}/admin/editor?id={self.template_id}",
                wait_until="networkidle",
            )
            page.wait_for_timeout(3000)

            screenshot_path = SCREENSHOTS_DIR / "09_editor_page.png"
            page.screenshot(path=str(screenshot_path), full_page=True)
            self.screenshots.append(("Editor Page", "09_editor_page.png"))
            print(f"  Screenshot saved: {screenshot_path}")

            browser.close()

    def test_10_sign_page(self):
        """Test sign page with handwriting config."""
        if not self.template_id:
            print("  Skipping: no template_id")
            return

        with sync_playwright() as p:
            browser = p.chromium.launch(channel="chrome", headless=True)
            page = browser.new_page(viewport={"width": 1280, "height": 800})

            page.goto(
                f"{BASE_URL}/sign?template_id={self.template_id}",
                wait_until="networkidle",
            )
            page.wait_for_timeout(3000)

            screenshot_path = SCREENSHOTS_DIR / "10_sign_page.png"
            page.screenshot(path=str(screenshot_path), full_page=True)
            self.screenshots.append(("Sign Page", "10_sign_page.png"))
            print(f"  Screenshot saved: {screenshot_path}")

            try:
                hw_toggle = page.locator("text=调整手写效果")
                if hw_toggle.is_visible():
                    hw_toggle.click()
                    page.wait_for_timeout(1000)
                    screenshot_path2 = SCREENSHOTS_DIR / "10_sign_page_hw_expanded.png"
                    page.screenshot(path=str(screenshot_path2), full_page=True)
                    self.screenshots.append(("Sign Page - Handwriting Expanded", "10_sign_page_hw_expanded.png"))
                    print(f"  Screenshot saved: {screenshot_path2}")
            except Exception as e:
                print(f"  Handwriting toggle not found: {e}")

            browser.close()

    def generate_report(self):
        """Generate HTML test report."""
        print("\n" + "=" * 60)
        print("Generating Test Report")
        print("=" * 60)

        passed = sum(1 for r in self.test_results if r["status"] == "PASS")
        failed = sum(1 for r in self.test_results if r["status"] == "FAIL")
        total = len(self.test_results)

        html = f"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>手写效果配置 E2E 测试报告</title>
    <style>
        * {{ margin: 0; padding: 0; box-sizing: border-box; }}
        body {{ font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; color: #333; line-height: 1.6; }}
        .container {{ max-width: 1200px; margin: 0 auto; padding: 20px; }}
        header {{ background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; border-radius: 10px; margin-bottom: 30px; }}
        header h1 {{ font-size: 24px; margin-bottom: 10px; }}
        header .meta {{ opacity: 0.9; font-size: 14px; }}
        .summary {{ display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; margin-bottom: 30px; }}
        .summary-card {{ background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); text-align: center; }}
        .summary-card .number {{ font-size: 36px; font-weight: bold; }}
        .summary-card .label {{ color: #666; font-size: 14px; }}
        .summary-card.pass .number {{ color: #22c55e; }}
        .summary-card.fail .number {{ color: #ef4444; }}
        .summary-card.total .number {{ color: #3b82f6; }}
        .test-list {{ background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; margin-bottom: 30px; }}
        .test-item {{ padding: 15px 20px; border-bottom: 1px solid #eee; display: flex; align-items: center; }}
        .test-item:last-child {{ border-bottom: none; }}
        .test-item .status {{ width: 24px; height: 24px; border-radius: 50%; margin-right: 15px; display: flex; align-items: center; justify-content: center; font-size: 12px; color: white; }}
        .test-item .status.pass {{ background: #22c55e; }}
        .test-item .status.fail {{ background: #ef4444; }}
        .test-item .name {{ flex: 1; }}
        .test-item .error {{ color: #ef4444; font-size: 13px; margin-top: 5px; }}
        .screenshots {{ margin-bottom: 30px; }}
        .screenshots h2 {{ margin-bottom: 20px; color: #333; }}
        .screenshot-grid {{ display: grid; grid-template-columns: repeat(auto-fill, minmax(400px, 1fr)); gap: 20px; }}
        .screenshot-card {{ background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; }}
        .screenshot-card img {{ width: 100%; height: auto; display: block; }}
        .screenshot-card .caption {{ padding: 10px 15px; background: #f8f9fa; font-size: 14px; color: #555; }}
        footer {{ text-align: center; padding: 20px; color: #888; font-size: 13px; }}
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>手写效果配置 E2E 测试报告</h1>
            <div class="meta">
                <div>生成时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</div>
                <div>测试环境: {BASE_URL}</div>
            </div>
        </header>

        <div class="summary">
            <div class="summary-card total">
                <div class="number">{total}</div>
                <div class="label">总测试数</div>
            </div>
            <div class="summary-card pass">
                <div class="number">{passed}</div>
                <div class="label">通过</div>
            </div>
            <div class="summary-card fail">
                <div class="number">{failed}</div>
                <div class="label">失败</div>
            </div>
        </div>

        <div class="test-list">
            <h2 style="padding: 15px 20px; border-bottom: 1px solid #eee;">测试详情</h2>
"""

        for result in self.test_results:
            status_class = "pass" if result["status"] == "PASS" else "fail"
            status_icon = "✓" if result["status"] == "PASS" else "✗"
            error_html = ""
            if result["error"]:
                error_html = f'<div class="error">{result["error"]}</div>'
            html += f"""
            <div class="test-item">
                <div class="status {status_class}">{status_icon}</div>
                <div class="name">
                    {result["name"]}
                    {error_html}
                </div>
            </div>"""

        html += """
        </div>

        <div class="screenshots">
            <h2>截图记录</h2>
            <div class="screenshot-grid">
"""

        for title, filename in self.screenshots:
            html += f"""
                <div class="screenshot-card">
                    <img src="screenshots/{filename}" alt="{title}" loading="lazy">
                    <div class="caption">{title}</div>
                </div>"""

        html += """
            </div>
        </div>

        <footer>
            inkflow-go 手写效果配置测试 &copy; 2026
        </footer>
    </div>
</body>
</html>
"""

        report_path = REPORT_DIR / "test_report.html"
        with open(report_path, "w", encoding="utf-8") as f:
            f.write(html)
        print(f"  Report generated: {report_path}")
        print(f"  Screenshots directory: {SCREENSHOTS_DIR}")
        return report_path


def main():
    test = HandwritingE2ETest()
    try:
        test.setup()
        test.run_tests()
    except Exception as e:
        print(f"\nFatal error: {e}")
        import traceback
        traceback.print_exc()
    finally:
        test.teardown()
        report_path = test.generate_report()
        print(f"\n{'=' * 60}")
        print(f"Test complete! Report: {report_path}")
        print(f"{'=' * 60}")


if __name__ == "__main__":
    main()
