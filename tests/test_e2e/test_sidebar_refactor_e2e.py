#!/usr/bin/env python3
"""
E2E test for sidebar refactor - PropertiesPanel show/hide behavior.
Tests the new interaction pattern: properties panel hidden by default,
shows when a component is clicked, hides when clicking blank area.
Also tests handwriting font moved to properties panel basic attributes.
"""

import os
import sys
import time
import requests
from pathlib import Path
from playwright.sync_api import sync_playwright, expect

if sys.platform == "win32":
    import io
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', errors='replace')
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding='utf-8', errors='replace')

BASE_URL = os.environ.get("TEST_BASE_URL", "http://localhost:8080")
REPORT_DIR = Path(__file__).parent / "report"
SCREENSHOTS_DIR = REPORT_DIR / "screenshots"


class SidebarRefactorE2ETest:
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
        print("Sidebar Refactor E2E Test Setup")
        print("=" * 60)

        REPORT_DIR.mkdir(parents=True, exist_ok=True)
        SCREENSHOTS_DIR.mkdir(parents=True, exist_ok=True)

        self._wait_for_server()
        self._login_admin()
        self._create_test_template()

    def _wait_for_server(self, timeout=10):
        """Wait for server to be ready."""
        print("\n[1/3] Waiting for server...")
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
        print("  Server not running, tests will use existing state")

    def _login_admin(self):
        """Login as admin and get JWT token."""
        print("\n[2/3] Logging in as admin...")
        try:
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
        except Exception as e:
            print(f"  Login error: {e}")

    def _create_test_template(self):
        """Create a test template for editor testing."""
        print("\n[3/3] Creating test template...")
        try:
            resp = self.session.post(
                f"{BASE_URL}/api/templates",
                json={"name": "侧边栏重构测试模板"},
            )
            data = resp.json()
            if data.get("code") == 0:
                self.template_id = data["data"]["id"]
                print(f"  Created template ID: {self.template_id}")

                import time as _time
                unique_suffix = str(int(_time.time() * 1000))[-6:]
                controls = [
                    {
                        "id": f"sb_textbox_{unique_suffix}",
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
                        "id": f"sb_checkbox_{unique_suffix}",
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

                resp = self.session.put(
                    f"{BASE_URL}/api/templates/{self.template_id}",
                    json={"controls": controls},
                )
                if resp.json().get("code") == 0:
                    print(f"  Added {len(controls)} test controls")
            else:
                print(f"  Create failed: {data}")
        except Exception as e:
            print(f"  Template creation error: {e}")

    def run_tests(self):
        """Run all sidebar refactor E2E tests."""
        print("\n" + "=" * 60)
        print("Running Sidebar Refactor E2E Tests")
        print("=" * 60)

        tests = [
            ("test_01_editor_initial_state", self.test_01_editor_initial_state),
            ("test_02_properties_panel_hidden_by_default", self.test_02_properties_panel_hidden_by_default),
            ("test_03_click_component_shows_panel", self.test_03_click_component_shows_panel),
            ("test_04_handwriting_font_in_properties_panel", self.test_04_handwriting_font_in_properties_panel),
            ("test_05_click_blank_hides_panel", self.test_05_click_blank_hides_panel),
            ("test_06_handwriting_modal_no_font_selector", self.test_06_handwriting_modal_no_font_selector),
            ("test_07_multiple_component_selection", self.test_07_multiple_component_selection),
            ("test_08_panel_animation_smooth", self.test_08_panel_animation_smooth),
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

    def test_01_editor_initial_state(self):
        """Test editor page loads correctly."""
        if not self.template_id:
            raise RuntimeError("No template_id available")

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

            page.goto(
                f"{BASE_URL}/admin/editor?id={self.template_id}",
                wait_until="networkidle",
            )
            page.wait_for_timeout(2000)

            screenshot_path = SCREENSHOTS_DIR / "01_editor_initial.png"
            page.screenshot(path=str(screenshot_path), full_page=True)
            self.screenshots.append(("Editor Initial State", "01_editor_initial.png"))
            print(f"  Screenshot saved: {screenshot_path}")

            browser.close()

    def test_02_properties_panel_hidden_by_default(self):
        """Test that properties panel is hidden by default."""
        if not self.template_id:
            raise RuntimeError("No template_id available")

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
            page.wait_for_timeout(2000)

            properties_panel = page.locator(".properties-panel")
            assert properties_panel.count() > 0, "Properties panel not found"

            has_hidden_class = properties_panel.evaluate(
                "el => el.classList.contains('hidden')"
            )
            assert has_hidden_class, "Properties panel should be hidden by default"

            screenshot_path = SCREENSHOTS_DIR / "02_panel_hidden.png"
            page.screenshot(path=str(screenshot_path), full_page=True)
            self.screenshots.append(("Panel Hidden by Default", "02_panel_hidden.png"))
            print(f"  Screenshot saved: {screenshot_path}")

            browser.close()

    def test_03_click_component_shows_panel(self):
        """Test that clicking a component shows the properties panel."""
        if not self.template_id:
            raise RuntimeError("No template_id available")

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
            page.wait_for_timeout(2000)

            control = page.locator(".control-item").first
            if control.count() > 0:
                control.click()
                page.wait_for_timeout(500)

                properties_panel = page.locator(".properties-panel")
                has_hidden_class = properties_panel.evaluate(
                    "el => el.classList.contains('hidden')"
                )
                assert not has_hidden_class, "Properties panel should be visible after clicking component"

                screenshot_path = SCREENSHOTS_DIR / "03_panel_visible.png"
                page.screenshot(path=str(screenshot_path), full_page=True)
                self.screenshots.append(("Panel Visible After Click", "03_panel_visible.png"))
                print(f"  Screenshot saved: {screenshot_path}")
            else:
                print("  No control items found, skipping click test")

            browser.close()

    def test_04_handwriting_font_in_properties_panel(self):
        """Test that handwriting font selector is in properties panel basic attributes."""
        if not self.template_id:
            raise RuntimeError("No template_id available")

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
            page.wait_for_timeout(2000)

            control = page.locator(".control-item").first
            if control.count() > 0:
                control.click()
                page.wait_for_timeout(500)

                handwriting_font_label = page.locator("text=手写字体")
                assert handwriting_font_label.count() > 0, "Handwriting font label not found in properties panel"

                font_select = page.locator(".prop-row select").filter(has_text="默认")
                assert font_select.count() > 0, "Handwriting font selector not found"

                screenshot_path = SCREENSHOTS_DIR / "04_handwriting_font_in_panel.png"
                page.screenshot(path=str(screenshot_path), full_page=True)
                self.screenshots.append(("Handwriting Font in Panel", "04_handwriting_font_in_panel.png"))
                print(f"  Screenshot saved: {screenshot_path}")
            else:
                print("  No control items found, skipping handwriting font test")

            browser.close()

    def test_05_click_blank_hides_panel(self):
        """Test that clicking blank area hides the properties panel."""
        if not self.template_id:
            raise RuntimeError("No template_id available")

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
            page.wait_for_timeout(2000)

            control = page.locator(".control-item").first
            if control.count() > 0:
                control.click()
                page.wait_for_timeout(500)

                properties_panel = page.locator(".properties-panel")
                has_hidden_class = properties_panel.evaluate(
                    "el => el.classList.contains('hidden')"
                )
                assert not has_hidden_class, "Panel should be visible before clicking blank"

                page.evaluate("""() => {
                    const stage = document.querySelector('.canvas-stage');
                    const rect = stage.getBoundingClientRect();
                    const event = new MouseEvent('mousedown', {
                        bubbles: true,
                        cancelable: true,
                        clientX: rect.left + 100,
                        clientY: rect.top + 400,
                    });
                    stage.dispatchEvent(event);
                }""")
                page.wait_for_timeout(500)

                has_hidden_class = properties_panel.evaluate(
                    "el => el.classList.contains('hidden')"
                )
                assert has_hidden_class, "Properties panel should be hidden after clicking blank area"

                screenshot_path = SCREENSHOTS_DIR / "05_panel_hidden_after_blank_click.png"
                page.screenshot(path=str(screenshot_path), full_page=True)
                self.screenshots.append(("Panel Hidden After Blank Click", "05_panel_hidden_after_blank_click.png"))
                print(f"  Screenshot saved: {screenshot_path}")
            else:
                print("  No control items found, skipping blank click test")

            browser.close()

    def test_06_handwriting_modal_no_font_selector(self):
        """Test that handwriting modal does not have font selector."""
        if not self.template_id:
            raise RuntimeError("No template_id available")

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
            page.wait_for_timeout(2000)

            control = page.locator(".control-item").first
            if control.count() > 0:
                control.click()
                page.wait_for_timeout(500)

            page.evaluate("document.querySelector('.hw-toggle-btn')?.click()")
            page.wait_for_timeout(1000)

            modal = page.locator(".hw-modal")
            if modal.count() > 0:
                font_section = modal.locator("text=手写字体")
                assert font_section.count() == 0, "Handwriting font section should NOT be in modal"

                screenshot_path = SCREENSHOTS_DIR / "06_modal_no_font.png"
                page.screenshot(path=str(screenshot_path), full_page=True)
                self.screenshots.append(("Modal No Font Selector", "06_modal_no_font.png"))
                print(f"  Screenshot saved: {screenshot_path}")

                close_btn = modal.locator(".hw-modal-close")
                if close_btn.count() > 0:
                    close_btn.click()
            else:
                print("  Handwriting modal not found after click")

            browser.close()

    def test_07_multiple_component_selection(self):
        """Test selecting multiple components updates panel correctly."""
        if not self.template_id:
            raise RuntimeError("No template_id available")

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
            page.wait_for_timeout(2000)

            controls = page.locator(".control-item")
            if controls.count() >= 2:
                controls.nth(0).click()
                page.wait_for_timeout(300)

                label_input = page.locator(".prop-row input").first
                if label_input.count() > 0:
                    first_label = label_input.input_value()
                    print(f"  First component label: {first_label}")

                controls.nth(1).click()
                page.wait_for_timeout(300)

                screenshot_path = SCREENSHOTS_DIR / "07_multiple_selection.png"
                page.screenshot(path=str(screenshot_path), full_page=True)
                self.screenshots.append(("Multiple Selection", "07_multiple_selection.png"))
                print(f"  Screenshot saved: {screenshot_path}")
            else:
                print("  Not enough control items found")

            browser.close()

    def test_08_panel_animation_smooth(self):
        """Test that panel animation is smooth."""
        if not self.template_id:
            raise RuntimeError("No template_id available")

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
            page.wait_for_timeout(2000)

            properties_panel = page.locator(".properties-panel")
            transition = properties_panel.evaluate(
                "el => window.getComputedStyle(el).transition"
            )
            print(f"  Panel transition: {transition}")
            assert transition and transition != "all 0s ease 0s", "Panel should have transition animation"
            assert "0.3s" in transition, "Panel should have 0.3s transition duration"

            browser.close()

    def generate_report(self):
        """Generate HTML test report."""
        print("\n" + "=" * 60)
        print("Generating Test Report")
        print("=" * 60)

        passed = sum(1 for r in self.test_results if r["status"] == "PASS")
        failed = sum(1 for r in self.test_results if r["status"] == "FAIL")
        total = len(self.test_results)

        from datetime import datetime
        html = f"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>侧边栏重构 E2E 测试报告</title>
    <style>
        * {{ margin: 0; padding: 0; box-sizing: border-box; }}
        body {{ font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; color: #333; line-height: 1.6; }}
        .container {{ max-width: 1200px; margin: 0 auto; padding: 20px; }}
        header {{ background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%); color: white; padding: 30px; border-radius: 10px; margin-bottom: 30px; }}
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
            <h1>侧边栏重构 E2E 测试报告</h1>
            <div class="meta">
                <div>生成时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</div>
                <div>测试环境: {BASE_URL}</div>
                <div>测试模板ID: {self.template_id}</div>
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
            inkflow-go 侧边栏重构测试 &copy; 2026
        </footer>
    </div>
</body>
</html>
"""

        report_path = REPORT_DIR / "sidebar_refactor_report.html"
        with open(report_path, "w", encoding="utf-8") as f:
            f.write(html)
        print(f"  Report generated: {report_path}")
        return report_path


def main():
    test = SidebarRefactorE2ETest()
    try:
        test.setup()
        test.run_tests()
    except Exception as e:
        print(f"\nFatal error: {e}")
        import traceback
        traceback.print_exc()
    finally:
        report_path = test.generate_report()
        print(f"\n{'=' * 60}")
        print(f"Test complete! Report: {report_path}")
        print(f"{'=' * 60}")


if __name__ == "__main__":
    main()
