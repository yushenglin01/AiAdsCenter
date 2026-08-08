from pathlib import Path

from playwright.sync_api import sync_playwright


BASE_URL = "http://localhost:5173"
SCREENSHOT = Path("/tmp/game-ads-stage2-imports.png")


with sync_playwright() as playwright:
    browser = playwright.chromium.launch(headless=True)
    page = browser.new_page(viewport={"width": 1440, "height": 960})
    console_errors: list[str] = []
    page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)

    page.goto(BASE_URL)
    page.wait_for_load_state("networkidle")
    page.get_by_label("用户名").fill("admin")
    page.get_by_label("密码").fill("Demo@123456")
    page.get_by_role("button", name="进入平台").click()
    page.wait_for_url(f"{BASE_URL}/")

    page.get_by_role("link", name="游戏管理").click()
    page.wait_for_url(f"{BASE_URL}/games")
    page.get_by_text("Galaxy Adventure").wait_for()

    page.get_by_role("link", name="投放资产").click()
    page.wait_for_url(f"{BASE_URL}/assets")
    page.get_by_text("Meta US Growth").first.wait_for()
    assert page.get_by_text("Google JP Stable").first.is_visible()

    page.get_by_role("link", name="数据导入").click()
    page.wait_for_url(f"{BASE_URL}/imports")
    page.get_by_text("meta_ad_metrics.json").wait_for()
    assert page.get_by_role("heading", name="数据导入").is_visible()
    assert page.get_by_text("meta_ad_metrics.json").is_visible()
    assert page.get_by_text("第 1 行：date 必须为 YYYY-MM-DD").is_visible()
    assert not console_errors, f"browser console errors: {console_errors}"
    page.screenshot(path=str(SCREENSHOT), full_page=True)
    print(f"stage2_e2e=ok screenshot={SCREENSHOT}")
    browser.close()
