from __future__ import annotations

import importlib.util
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SCHEDULER_PATH = ROOT / ".github" / "workflows" / "scripts" / "crane_scheduler.py"

spec = importlib.util.spec_from_file_location("crane_scheduler", SCHEDULER_PATH)
assert spec is not None
crane_scheduler = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(crane_scheduler)


def test_completed_state_skips_inactive_migration() -> None:
    should_skip, reason = crane_scheduler.check_skip_conditions({"completed": True})

    assert should_skip is True
    assert reason == "completed: target metric reached"


def test_active_issue_overrides_stale_completed_state() -> None:
    should_skip, reason = crane_scheduler.check_skip_conditions(
        {"completed": True},
        issue_active=True,
    )

    assert should_skip is False
    assert reason is None


def test_active_issue_does_not_override_pause() -> None:
    should_skip, reason = crane_scheduler.check_skip_conditions(
        {"completed": True, "paused": True, "pause_reason": "manual hold"},
        issue_active=True,
    )

    assert should_skip is True
    assert reason == "paused: manual hold"


def test_machine_state_completed_string_is_recognized() -> None:
    assert crane_scheduler.is_completed_state({"completed": "true"}) is True
