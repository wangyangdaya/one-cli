from __future__ import annotations

import os
import sys
import types
from pathlib import Path
from unittest.mock import patch

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

filesystem_module = types.ModuleType("deepagents.backends.filesystem")


class DummyFilesystemBackend:
    def __init__(self, *args, **kwargs) -> None:
        pass


filesystem_module.FilesystemBackend = DummyFilesystemBackend
protocol_module = types.ModuleType("deepagents.backends.protocol")
protocol_module.ExecuteResponse = type("ExecuteResponse", (), {})
protocol_module.SandboxBackendProtocol = object
sys.modules.setdefault("deepagents", types.ModuleType("deepagents"))
sys.modules.setdefault("deepagents.backends", types.ModuleType("deepagents.backends"))
sys.modules["deepagents.backends.filesystem"] = filesystem_module
sys.modules["deepagents.backends.protocol"] = protocol_module

import main


def read_env_file(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    if not path.exists():
        return values

    for raw_line in path.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip()
        value = value.strip().strip('"').strip("'")
        values[key] = value
    return values


def test_resolve_app_dir_defaults_to_repo_tmp_openapi(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("SKILLS_VERIFY_APP_DIR", raising=False)
    repo_root = Path("/repo")

    assert main.resolve_app_dir(repo_root) == repo_root / "tmp" / "openapi"


def test_resolve_app_dir_uses_relative_env_value(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("SKILLS_VERIFY_APP_DIR", "tmp/custom-app")
    repo_root = Path("/repo")

    assert main.resolve_app_dir(repo_root) == repo_root / "tmp" / "custom-app"


def test_resolve_app_dir_preserves_absolute_env_value(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("SKILLS_VERIFY_APP_DIR", "/tmp/generated-app")
    repo_root = Path("/repo")

    assert main.resolve_app_dir(repo_root) == Path("/tmp/generated-app")


def test_build_agent_uses_resolved_app_dir_for_skills_and_backend(tmp_path: Path) -> None:
    app_dir = tmp_path / "generated-app"
    skills_dir = app_dir / "skills"
    skills_dir.mkdir(parents=True)

    backend_calls: list[dict] = []

    class FakeMemorySaver:
        pass

    def fake_backend(*, repo_root, app_dir, executables=None):
        backend_calls.append(
            {"repo_root": repo_root, "app_dir": app_dir, "executables": executables}
        )
        return object()

    def fake_create_deep_agent(**kwargs):
        return kwargs

    deepagents_module = sys.modules["deepagents"]
    previous_create_deep_agent = getattr(deepagents_module, "create_deep_agent", None)
    deepagents_module.create_deep_agent = fake_create_deep_agent

    memory_module = types.ModuleType("langgraph.checkpoint.memory")
    memory_module.MemorySaver = FakeMemorySaver
    sys.modules.setdefault("langgraph", types.ModuleType("langgraph"))
    sys.modules.setdefault("langgraph.checkpoint", types.ModuleType("langgraph.checkpoint"))
    sys.modules["langgraph.checkpoint.memory"] = memory_module

    try:
        with patch.object(main, "build_llm", return_value="llm"), patch.object(
            main, "resolve_repo_root", return_value=tmp_path
        ), patch.object(main, "resolve_app_dir", return_value=app_dir), patch.object(
            main, "CliBackend", side_effect=fake_backend
        ):
            agent = main.build_agent()
    finally:
        if previous_create_deep_agent is None:
            delattr(deepagents_module, "create_deep_agent")
        else:
            deepagents_module.create_deep_agent = previous_create_deep_agent

    assert backend_calls == [
        {"repo_root": tmp_path, "app_dir": app_dir, "executables": "openapi-cli"}
    ]
    assert agent["skills"] == [str(skills_dir)]


@pytest.mark.anyio
async def test_invoke_agent_returns_success_result() -> None:
    class FakeAgent:
        async def ainvoke(self, payload, config=None):
            return {
                "messages": [
                    type(
                        "Message",
                        (),
                        {"content": "assistant", "usage_metadata": {"input_tokens": 2}},
                    )()
                ]
            }

    result = await main.invoke_agent(
        agent=FakeAgent(),
        messages=[{"role": "user", "content": "hello"}],
        config={"configurable": {"thread_id": "demo"}},
    )

    assert main.extract_text(result) == "assistant"


@pytest.mark.anyio
async def test_invoke_agent_propagates_failure() -> None:
    class FakeAgent:
        async def ainvoke(self, payload, config=None):
            raise RuntimeError("boom")

    with pytest.raises(RuntimeError, match="boom"):
        await main.invoke_agent(
            agent=FakeAgent(),
            messages=[{"role": "user", "content": "hello"}],
            config={"configurable": {"thread_id": "demo"}},
        )


@pytest.mark.anyio
async def test_run_repl_uses_resolved_paths(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    app_dir = tmp_path / "generated-app"
    skills_dir = app_dir / "skills"
    skills_dir.mkdir(parents=True)

    class FakeAgent:
        pass

    async def fake_invoke_agent(**kwargs):
        return {"messages": [type("Message", (), {"content": "assistant"})()]}

    inputs = iter(["hello", "quit"])
    monkeypatch.setenv("LLM_MODEL_NAME", "gpt-test")

    with patch.object(main, "resolve_repo_root", return_value=tmp_path), patch.object(
        main, "resolve_app_dir", return_value=app_dir
    ), patch.object(main, "build_agent", return_value=FakeAgent()), patch.object(
        main, "get_all_skills", return_value={str(skills_dir / "SKILL.md"): "demo"}
    ), patch.object(main, "invoke_agent", side_effect=fake_invoke_agent), patch(
        "builtins.input", side_effect=lambda _: next(inputs)
    ):
        assert await main.run_repl("demo") == 0


@pytest.mark.anyio
async def test_dump_quark_web_search_tools() -> None:
    pytest.importorskip("langchain_mcp_adapters")
    from langchain_mcp_adapters.client import MultiServerMCPClient

    repo_root = Path(__file__).resolve().parents[1]
    env_values = read_env_file(repo_root / ".env")

    mcp_key = os.getenv("MCP_KEY") or os.getenv("QUARK_MCP_KEY") or env_values.get(
        "MCP_KEY"
    ) or env_values.get("QUARK_MCP_KEY")
    if not mcp_key:
        pytest.skip("Set MCP_KEY or QUARK_MCP_KEY to dump the Quark MCP tools list.")

    client = MultiServerMCPClient(
        {
            "tool-quark-web-search": {
                "transport": "streamable_http",
                "url": os.getenv("QUARK_SEARCH_MCP_URL")
                or env_values.get("QUARK_SEARCH_MCP_URL")
                or "https://tool-quh-tdsunr-rlwpdeagti.cn-hangzhou.fcapp.run/mcp",
                "headers": {"Authorization": f"Bearer {mcp_key}"},
            }
        }
    )

    tools = await client.get_tools()

    tool_names = [getattr(tool, "name", str(tool)) for tool in tools]
    print("MCP tools:")
    for tool_name in tool_names:
        print(f"- {tool_name}")

    assert tool_names
