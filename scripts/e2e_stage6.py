import os
from datetime import date, timedelta
from pathlib import Path

from playwright.sync_api import sync_playwright


BASE_URL = "http://localhost:5173"
API_URL = f"{BASE_URL}/api/v1"
SCREENSHOT = Path("/tmp/game-ads-stage6-approval-center.png")
TERMINAL = {"WAITING_APPROVAL", "SUCCEEDED", "FAILED", "MANUAL_REVIEW", "CANCELLED"}
SYSTEM_AGENT_ID = "20000000-0000-4000-8000-000000000006"
OPERATOR_ID = "20000000-0000-4000-8000-000000000003"


def login(request, username: str) -> str:
    response = request.post(f"{API_URL}/auth/login", data={"username": username, "password": "Demo@123456"})
    assert response.ok, f"{username} login failed: {response.status}"
    return response.json()["data"]["access_token"]


def auth(token: str) -> dict[str, str]:
    return {"Authorization": f"Bearer {token}"}


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

    operator_token = login(page.request, "operator")
    manager_token = login(page.request, "manager")
    admin_token = login(page.request, "admin")
    system_login = page.request.post(f"{API_URL}/auth/login", data={"username": "system-agent", "password": "Demo@123456"})
    assert system_login.status == 401, "SYSTEM_AGENT unexpectedly received a login token"

    campaigns_response = page.request.get(f"{API_URL}/campaigns", headers=auth(operator_token))
    assert campaigns_response.ok
    campaign = next(item for item in campaigns_response.json()["data"] if item["name"] == "Meta US Growth")

    task_id = ""
    final_details = None
    for offset in range(20):
        analysis_date = (date(2026, 8, 12) + timedelta(days=offset)).isoformat()
        submitted = page.request.post(f"{API_URL}/analysis/business", headers=auth(operator_token), data={"game_id": campaign["game_id"], "campaign_id": campaign["id"], "analysis_date": analysis_date})
        assert submitted.status == 202
        task_id = submitted.json()["data"]["task"]["task_id"]
        for _ in range(60):
            response = page.request.get(f"{API_URL}/analysis/business/tasks/{task_id}", headers=auth(operator_token))
            assert response.ok
            final_details = response.json()["data"]
            if final_details["task"]["status"] in TERMINAL:
                break
            page.wait_for_timeout(250)
        if final_details and final_details["task"]["status"] == "WAITING_APPROVAL" and final_details["task"]["created_by"] == OPERATOR_ID:
            break
    assert final_details is not None
    assert final_details["task"]["status"] == "WAITING_APPROVAL", final_details["task"]
    assert final_details["report"]["status"] == "READY"
    assert len(final_details["approvals"]) == 2
    assert all(row["requested_by"] == SYSTEM_AGENT_ID for row in final_details["approvals"])
    assert all(row["report_id"] == final_details["report"]["id"] for row in final_details["approvals"])

    approvals_response = page.request.get(f"{API_URL}/approvals", headers=auth(operator_token), params={"status": "PENDING"})
    assert approvals_response.ok
    approvals = [row for row in approvals_response.json()["data"] if row["approval"]["task_id"] == task_id]
    assert len(approvals) == 2

    denied = page.request.post(f"{API_URL}/approvals/{approvals[0]['approval']['id']}/approve", headers=auth(operator_token), data={"comment": "operator must not approve"})
    assert denied.status == 403, f"OPERATOR approval returned {denied.status}"

    approved = page.request.post(f"{API_URL}/approvals/{approvals[0]['approval']['id']}/approve", headers=auth(manager_token), data={"comment": "指标证据充分，同意进入人工执行评估"})
    assert approved.ok
    assert approved.json()["data"]["approval"]["status"] == "APPROVED"
    rejected = page.request.post(f"{API_URL}/approvals/{approvals[1]['approval']['id']}/reject", headers=auth(manager_token), data={"comment": "先补充新素材排期再处理"})
    assert rejected.ok
    assert rejected.json()["data"]["approval"]["status"] == "REJECTED"

    task_response = page.request.get(f"{API_URL}/analysis/business/tasks/{task_id}", headers=auth(manager_token))
    assert task_response.ok
    completed = task_response.json()["data"]
    assert completed["task"]["status"] == "SUCCEEDED"
    assert {row["status"] for row in completed["approvals"]} == {"APPROVED", "REJECTED"}
    assert {row["status"] for row in completed["recommendations"] if row["requires_approval"]} == {"APPROVED", "REJECTED"}

    manager_audit = page.request.get(f"{API_URL}/audit-logs", headers=auth(manager_token))
    assert manager_audit.status == 403
    audit_response = page.request.get(f"{API_URL}/audit-logs", headers=auth(admin_token), params={"task_id": task_id, "limit": 100})
    assert audit_response.ok
    audits = audit_response.json()["data"]
    actions = {row["action"] for row in audits}
    assert {"AGENT_TOOL_CALL", "APPROVAL_DECISION_DENIED", "APPROVAL_APPROVED", "APPROVAL_REJECTED"}.issubset(actions), actions
    assert len([row for row in audits if row["action"] == "AGENT_TOOL_CALL"]) >= 7
    decisions = [row for row in audits if row["action"] in {"APPROVAL_APPROVED", "APPROVAL_REJECTED"}]
    assert all(row["metadata"]["advertising_platform_called"] is False for row in decisions)

    usage = page.request.get(f"{API_URL}/model-usage/summary", headers=auth(manager_token))
    assert usage.ok and usage.json()["data"]["calls"] > 0
    dashboard = page.request.get(f"{API_URL}/dashboard/operations", headers=auth(operator_token))
    assert dashboard.ok and dashboard.json()["data"]["audit_events"] >= len(audits)

    page.goto(BASE_URL)
    page.wait_for_load_state("networkidle")
    page.get_by_label("用户名").fill("manager")
    page.get_by_label("密码").fill("Demo@123456")
    page.get_by_role("button", name="进入平台").click()
    page.wait_for_url(f"{BASE_URL}/")
    page.get_by_role("link", name="审批中心").click()
    page.wait_for_url(f"{BASE_URL}/approvals")
    page.wait_for_load_state("networkidle")
    page.get_by_role("heading", name="审批中心").wait_for()
    page.get_by_text("批准只更新建议状态").wait_for()
    page.get_by_text("PHASE 06").wait_for()
    assert not console_errors, f"browser console errors: {console_errors}"
    page.screenshot(path=str(SCREENSHOT), full_page=True)
    print(f"stage6_e2e=ok task={task_id} approvals=2 audit_events={len(audits)} screenshot={SCREENSHOT}")
    browser.close()
