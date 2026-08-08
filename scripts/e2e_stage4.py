import os
from pathlib import Path

from playwright.sync_api import sync_playwright


BASE_URL = "http://localhost:5173"
SCREENSHOT = Path("/tmp/game-ads-stage4-business-analysis.png")


with sync_playwright() as playwright:
    executable_path = os.getenv("PLAYWRIGHT_CHROMIUM_EXECUTABLE")
    launch_options = {"headless": True}
    if executable_path:
        launch_options["executable_path"] = executable_path
    browser = playwright.chromium.launch(**launch_options)
    page = browser.new_page(viewport={"width": 1520, "height": 1040})
    page.set_default_timeout(10_000)
    console_errors: list[str] = []
    page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)

    page.goto(BASE_URL)
    page.wait_for_load_state("networkidle")
    page.get_by_label("用户名").fill("admin")
    page.get_by_label("密码").fill("Demo@123456")
    page.get_by_role("button", name="进入平台").click()
    page.wait_for_url(f"{BASE_URL}/")

    page.get_by_role("link", name="经营分析").click()
    page.wait_for_url(f"{BASE_URL}/business-analysis")
    page.wait_for_load_state("networkidle")
    page.get_by_role("heading", name="经营分析").wait_for()
    page.get_by_text("默认使用 Mock LLM").wait_for()
    page.get_by_role("combobox", name="广告计划").click(force=True)
    page.get_by_role("option", name="Meta US Growth").click()
    page.locator(".analysis-launcher .el-select__selected-item", has_text="Meta US Growth").wait_for()
    assert page.get_by_role("button", name="发起经营分析").is_enabled()

    task_count_before = page.locator(".task-list button").count()
    page.get_by_role("button", name="发起经营分析").click()
    page.locator(".result-header .el-tag").filter(has_text="SUCCEEDED").or_(page.locator(".result-header .el-tag").filter(has_text="WAITING_APPROVAL")).first.wait_for(timeout=15_000)
    page.get_by_text("mock").first.wait_for()
    page.get_by_text("REDUCE_BUDGET").wait_for()
    page.get_by_text("REPLACE_CREATIVE").wait_for()
    page.get_by_text("VERIFY_ATTRIBUTION").wait_for()
    assert page.get_by_text("等待人工审批").count() == 2
    assert page.get_by_text("EXECUTED").count() == 0

    task_count_after_first = page.locator(".task-list button").count()
    assert task_count_after_first == task_count_before + 1 or task_count_after_first == task_count_before
    page.get_by_role("button", name="发起经营分析").click()
    page.locator(".result-header .el-tag").filter(has_text="SUCCEEDED").or_(page.locator(".result-header .el-tag").filter(has_text="WAITING_APPROVAL")).first.wait_for(timeout=15_000)
    assert page.locator(".task-list button").count() == task_count_after_first, "duplicate analysis created another task"

    page.locator(".task-list button", has_text="Google JP Stable").click()
    page.get_by_text("没有经营风险发现").wait_for()
    assert page.get_by_text("等待人工审批").count() == 0, "healthy control generated an approval"
    page.locator(".task-list button", has_text="Meta US Growth").first.click()
    page.get_by_text("REDUCE_BUDGET").wait_for()

    assert not console_errors, f"browser console errors: {console_errors}"
    page.screenshot(path=str(SCREENSHOT), full_page=True)
    print(f"stage4_e2e=ok tasks={task_count_after_first} screenshot={SCREENSHOT}")
    browser.close()
