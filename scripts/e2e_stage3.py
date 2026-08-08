from pathlib import Path

from playwright.sync_api import sync_playwright


BASE_URL = "http://localhost:5173"
SCREENSHOT = Path("/tmp/game-ads-stage3-dashboard.png")


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
    page.get_by_role("heading", name="经营总览").wait_for()
    page.get_by_text("D7 收入").wait_for()
    page.locator(".trend-chart canvas").wait_for()
    page.screenshot(path=str(SCREENSHOT), full_page=True)

    page.get_by_role("link", name="计划指标").click()
    page.wait_for_url(f"{BASE_URL}/metrics")
    page.get_by_text("Meta US Growth").wait_for()
    page.get_by_text("Google JP Stable").wait_for()
    page.get_by_text("TikTok KR Scale").wait_for()

    page.get_by_role("link", name="归因分析").click()
    page.wait_for_url(f"{BASE_URL}/attribution")
    page.get_by_text("安装归因存在偏差").wait_for()
    page.get_by_text("Meta US Growth").wait_for()

    page.get_by_role("link", name="素材分析").click()
    page.wait_for_url(f"{BASE_URL}/creative-analysis")
    page.get_by_text("素材疲劳风险高").wait_for()
    page.get_by_text("Galaxy Launch Video 302").first.wait_for()

    page.get_by_role("link", name="规则中心").click()
    page.wait_for_url(f"{BASE_URL}/rules")
    page.get_by_text("ROAS_BELOW_TARGET").first.wait_for()
    assert page.get_by_text("Google JP Stable").count() == 0, "healthy Google control should not appear in findings"

    assert not console_errors, f"browser console errors: {console_errors}"
    print(f"stage3_e2e=ok screenshot={SCREENSHOT}")
    browser.close()
