import os
from pathlib import Path

from playwright.sync_api import sync_playwright


BASE_URL = "http://localhost:5173"
API_URL = f"{BASE_URL}/api/v1"
SCREENSHOT = Path("/tmp/game-ads-stage5-async-report.png")
ANALYSIS_DATE = "2026-08-09"
TERMINAL = {"WAITING_APPROVAL", "SUCCEEDED", "FAILED", "MANUAL_REVIEW", "CANCELLED"}


with sync_playwright() as playwright:
    executable_path = os.getenv("PLAYWRIGHT_CHROMIUM_EXECUTABLE")
    launch_options = {"headless": True}
    if executable_path:
        launch_options["executable_path"] = executable_path
    browser = playwright.chromium.launch(**launch_options)
    page = browser.new_page(viewport={"width": 1520, "height": 1040})
    page.set_default_timeout(15_000)
    console_errors: list[str] = []
    page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)

    page.goto(BASE_URL)
    page.wait_for_load_state("networkidle")
    page.get_by_label("用户名").fill("admin")
    page.get_by_label("密码").fill("Demo@123456")
    page.get_by_role("button", name="进入平台").click()
    page.wait_for_url(f"{BASE_URL}/")

    token = page.evaluate("localStorage.getItem('gai_access_token')")
    headers = {"Authorization": f"Bearer {token}"}
    campaigns_response = page.request.get(f"{API_URL}/campaigns", headers=headers)
    assert campaigns_response.ok
    campaigns = campaigns_response.json()["data"]
    campaign = next(item for item in campaigns if item["name"] == "Meta US Growth")

    payload = {
        "game_id": campaign["game_id"],
        "campaign_id": campaign["id"],
        "analysis_date": ANALYSIS_DATE,
    }
    submitted = page.request.post(f"{API_URL}/analysis/business", headers=headers, data=payload)
    assert submitted.status == 202, f"expected asynchronous 202, got {submitted.status}"
    task = submitted.json()["data"]["task"]
    task_id = task["task_id"]

    duplicate = page.request.post(f"{API_URL}/analysis/business", headers=headers, data=payload)
    assert duplicate.status == 202
    assert duplicate.json()["data"]["task"]["task_id"] == task_id, "idempotent submit created another task"

    final_details = None
    for _ in range(40):
        task_response = page.request.get(f"{API_URL}/analysis/business/tasks/{task_id}", headers=headers)
        assert task_response.ok
        final_details = task_response.json()["data"]
        if final_details["task"]["status"] in TERMINAL:
            break
        page.wait_for_timeout(250)
    assert final_details is not None
    assert final_details["task"]["status"] in {"WAITING_APPROVAL", "SUCCEEDED"}, final_details["task"]
    assert final_details["task"]["current_step"] in {"APPROVAL_CREATION", "COMPLETED"}
    assert final_details["report"]["status"] == "READY"
    assert "本报告只包含建议，不代表广告平台动作已执行" in final_details["report"]["content_markdown"]

    report_response = page.request.get(f"{API_URL}/analysis/business/tasks/{task_id}/report", headers=headers)
    assert report_response.ok
    assert report_response.json()["data"]["task_id"] == task_id

    events_response = page.request.get(f"{API_URL}/analysis/business/tasks/{task_id}/events", headers=headers)
    assert events_response.ok
    assert "text/event-stream" in events_response.headers.get("content-type", "")
    assert "event:task" in events_response.text().replace(" ", "")

    page.goto(f"{BASE_URL}/business-analysis")
    page.wait_for_load_state("networkidle")
    page.get_by_role("heading", name="经营分析").wait_for()
    page.locator(".task-list button", has_text="Meta US Growth").first.click()
    page.get_by_text("REPORT SNAPSHOT").wait_for()
    page.get_by_role("button", name="下载报告").wait_for()
    page.locator(".result-header .el-tag").filter(has_text="SUCCEEDED").or_(page.locator(".result-header .el-tag").filter(has_text="WAITING_APPROVAL")).first.wait_for()
    assert page.get_by_text("EXECUTED").count() == 0
    assert not console_errors, f"browser console errors: {console_errors}"

    page.screenshot(path=str(SCREENSHOT), full_page=True)
    print(f"stage5_e2e=ok task={task_id} status={final_details['task']['status']} screenshot={SCREENSHOT}")
    browser.close()
