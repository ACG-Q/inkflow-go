import os
import pytest
from playwright.sync_api import sync_playwright

BASE_URL = os.environ.get("TEST_BASE_URL", "http://localhost:9876")


class TestSignFlow:
    def test_home_page_loads(self):
        with sync_playwright() as p:
            browser = p.chromium.launch(channel='chrome', headless=True)
            page = browser.new_page()
            page.goto(BASE_URL)
            assert page.title() is not None
            browser.close()

    def test_admin_login(self, api):
        with sync_playwright() as p:
            browser = p.chromium.launch(channel='chrome', headless=True)
            page = browser.new_page()
            page.goto(f"{BASE_URL}/admin/login", wait_until='networkidle')
            page.wait_for_selector("#app", timeout=10000)
            page.fill("input[type='text']", "admin", timeout=10000)
            page.fill("input[type='password']", "admin123", timeout=10000)
            page.click("button[type='submit']")
            page.wait_for_url(f"{BASE_URL}/admin", timeout=10000)
            assert "/admin" in page.url
            browser.close()
