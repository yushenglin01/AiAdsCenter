from pathlib import Path

from playwright.sync_api import sync_playwright


BASE_URL = "http://localhost:5173"
SCREENSHOT = Path("/tmp/game-ads-stage1-dashboard.png")


with sync_playwright() as playwright:
    browser = playwright.chromium.launch(headless=True)
    page = browser.new_page(viewport={"width": 1440, "height": 960})
    console_errors: list[str] = []
    page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)

    page.goto(BASE_URL)
    page.wait_for_load_state("networkidle")
    assert page.get_by_role("heading", name="登录演示环境").is_visible()
    assert page.get_by_role("button", name="进入平台").is_visible()

    page.get_by_label("用户名").fill("admin")
    page.get_by_label("密码").fill("Demo@123456")
    page.get_by_role("button", name="进入平台").click()
    page.wait_for_url(f"{BASE_URL}/")
    page.wait_for_load_state("networkidle")

    assert page.get_by_text("阶段二数据底座已就绪").is_visible()
    assert page.get_by_text("Demo Company").is_visible()
    assert page.get_by_text("Demo Admin").is_visible()
    assert page.get_by_role("link", name="系统管理").is_visible()
    assert not console_errors, f"browser console errors: {console_errors}"
    page.screenshot(path=str(SCREENSHOT), full_page=True)
    print(f"stage1_e2e=ok screenshot={SCREENSHOT}")
    browser.close()
