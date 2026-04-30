from __future__ import annotations

import argparse
import asyncio
import json
import logging
import os
from pathlib import Path

from langchain_core.stores import InMemoryStore

from cli_backend import CliBackend

SYSTEM_PROMPT = """You are validating generated opencli skills.
Always use the provided skills when they apply.
Only use commands that can be executed through the provided CLI backend.
Do not invent APIs or shell commands.
"""

DEFAULT_MODEL_NAME = "gpt-4o-mini"
DEFAULT_LOG_LEVEL = "INFO"
DEFAULT_THREAD_ID = "skills-verify"
DEFAULT_RECURSION_LIMIT = 25
DEFAULT_APP_DIR = "tmp/openapi"
DEFAULT_SKILL_LOG_PREVIEW_CHARS = 240
logger = logging.getLogger("skills_verify")


def resolve_repo_root() -> Path:
    return Path(__file__).resolve().parents[1]


def resolve_app_dir(repo_root: Path) -> Path:
    app_dir = Path(os.getenv("SKILLS_VERIFY_APP_DIR", DEFAULT_APP_DIR))
    if app_dir.is_absolute():
        return app_dir
    return repo_root / app_dir


def setup_logging() -> None:
    logging.basicConfig(
        level=getattr(logging, os.getenv("LOG_LEVEL", DEFAULT_LOG_LEVEL).upper(), logging.INFO),
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )


def load_environment() -> None:
    try:
        from dotenv import load_dotenv
    except ImportError as exc:  # pragma: no cover
        raise SystemExit(
            "python-dotenv is required. Install dependencies with `uv sync` in skills-verify/."
        ) from exc

    env_file = Path(__file__).with_name(".env")
    load_dotenv(env_file)


def build_llm():
    try:
        from langchain_openai import ChatOpenAI
    except ImportError as exc:  # pragma: no cover
        raise SystemExit(
            "langchain-openai is required. Install dependencies with `uv sync` in skills-verify/."
        ) from exc

    base_url = os.getenv("LLM_BASE_URL")
    api_key = os.getenv("LLM_API_KEY")
    model_name = os.getenv("LLM_MODEL_NAME", DEFAULT_MODEL_NAME)
    if not base_url:
        raise SystemExit("Set LLM_BASE_URL in skills-verify/.env.")
    if not api_key:
        raise SystemExit("Set LLM_API_KEY in skills-verify/.env.")

    return ChatOpenAI(
        model=model_name,
        stream_usage=True,
        temperature=0.3,
        base_url=base_url,
        api_key=api_key,
    )


def build_agent():
    try:
        from deepagents import create_deep_agent
    except ImportError as exc:  # pragma: no cover
        raise SystemExit(
            "deepagents is required. Install dependencies with `uv sync` in skills-verify/."
        ) from exc
    try:
        from langgraph.checkpoint.memory import MemorySaver
    except ImportError as exc:  # pragma: no cover
        raise SystemExit(
            "langgraph checkpoint support is required. Install dependencies with `uv sync` in skills-verify/."
        ) from exc

    repo_root = resolve_repo_root()
    app_dir = resolve_app_dir(repo_root)
    skills_dir = app_dir / "skills"
    if not skills_dir.exists():
        raise SystemExit(f"Skills directory not found: {skills_dir}")

    # Get allowed executables from environment variable
    executables = os.getenv("ALLOWED_EXECUTABLES", "openapi-cli")
    skills = [str(skills_dir)]

    store = InMemoryStore()
    logger.info(
        "agent.build model=%s executables=%s skills=%s",
        os.getenv("LLM_MODEL_NAME", DEFAULT_MODEL_NAME),
        executables,
        skills,
    )

    return create_deep_agent(
        model=build_llm(),
        backend=CliBackend(repo_root=repo_root, app_dir=app_dir, executables=executables),
        system_prompt=SYSTEM_PROMPT,
        skills=skills,
        checkpointer=MemorySaver(),
        name="openapi_skills_verifier",
        store=store,
        # permissions=[
        #     FilesystemPermission(
        #         operations=["write"],
        #         paths=["/**"],
        #         mode="deny",
        #     ),
        # ],
    )


def get_all_skills(skills_dir: Path) -> dict[str, str]:
    skills_files: dict[str, str] = {}
    for skill_file in sorted(skills_dir.rglob("SKILL.md")):
        skills_files[str(skill_file)] = skill_file.read_text(encoding="utf-8")
    return skills_files


def extract_text(result) -> str:
    if isinstance(result, dict):
        messages = result.get("messages", [])
        if messages:
            last = messages[-1]
            content = getattr(last, "content", last)
            return str(content)
    return str(result)


def _preview_text(value: str, *, max_chars: int = DEFAULT_SKILL_LOG_PREVIEW_CHARS) -> str:
    compact = " ".join(value.split())
    if len(compact) <= max_chars:
        return compact
    return f"{compact[:max_chars]}..."


def _to_log_text(value) -> str:
    if value is None:
        return ""
    if isinstance(value, str):
        return value
    try:
        return json.dumps(value, ensure_ascii=False)
    except TypeError:
        return str(value)


def log_skills_content(skills_files: dict[str, str]) -> None:
    logger.info("skills.loaded count=%d", len(skills_files))
    for skill_path in sorted(skills_files):
        content = skills_files[skill_path]
        logger.info(
            "skill.loaded path=%s chars=%d preview=%s",
            skill_path,
            len(content),
            _preview_text(content),
        )


def log_tool_events(result) -> None:
    if not isinstance(result, dict):
        return

    messages = result.get("messages", [])
    for message in messages:
        tool_calls = getattr(message, "tool_calls", None)
        if tool_calls is None and isinstance(message, dict):
            tool_calls = message.get("tool_calls")
        if tool_calls is None:
            additional_kwargs = getattr(message, "additional_kwargs", None)
            if isinstance(additional_kwargs, dict):
                tool_calls = additional_kwargs.get("tool_calls")

        if tool_calls:
            for tool_call in tool_calls:
                if not isinstance(tool_call, dict):
                    logger.info("tool.call raw=%s", _preview_text(_to_log_text(tool_call)))
                    continue

                function = tool_call.get("function") if isinstance(tool_call.get("function"), dict) else {}
                name = tool_call.get("name") or function.get("name") or "unknown"
                call_id = tool_call.get("id") or tool_call.get("tool_call_id") or ""
                args = tool_call.get("args")
                if args is None:
                    args = function.get("arguments")
                logger.info(
                    "tool.call name=%s id=%s args=%s",
                    name,
                    call_id,
                    _preview_text(_to_log_text(args)),
                )

        msg_type = getattr(message, "type", None)
        if msg_type is None and isinstance(message, dict):
            msg_type = message.get("type") or message.get("role")
        is_tool_message = msg_type == "tool" or message.__class__.__name__ == "ToolMessage"
        if not is_tool_message:
            continue

        tool_name = getattr(message, "name", None)
        if tool_name is None and isinstance(message, dict):
            tool_name = message.get("name")
        tool_call_id = getattr(message, "tool_call_id", None)
        if tool_call_id is None and isinstance(message, dict):
            tool_call_id = message.get("tool_call_id")
        content = getattr(message, "content", "")
        if isinstance(message, dict):
            content = message.get("content", "")

        logger.info(
            "tool.result name=%s id=%s content=%s",
            tool_name or "unknown",
            tool_call_id or "",
            _preview_text(_to_log_text(content)),
        )


async def invoke_agent(
        *,
        agent,
        messages: list[dict[str, str]],
        config: dict,
):
    thread_id = config.get("configurable", {}).get("thread_id")
    recursion_limit = config.get("recursion_limit")
    last_message = messages[-1] if messages else {}
    logger.info(
        "agent.invoke thread_id=%s recursion_limit=%s messages_count=%d last_role=%s last_content=%s",
        thread_id,
        recursion_limit,
        len(messages),
        last_message.get("role"),
        _preview_text(str(last_message.get("content", ""))),
    )
    return await agent.ainvoke({"messages": messages}, config=config)


def build_agent_config(thread_id: str) -> dict:
    return {
        "configurable": {
            "thread_id": thread_id,
        },
        "recursion_limit": DEFAULT_RECURSION_LIMIT,
    }


async def run_repl(thread_id: str) -> int:
    repo_root = resolve_repo_root()
    app_dir = resolve_app_dir(repo_root)
    skills_dir = app_dir / "skills"
    if not skills_dir.exists():
        raise SystemExit(f"Skills directory not found: {skills_dir}")

    print("正在初始化 openapi skills verifier（请稍候）...", flush=True)
    print(f"skills 目录: {skills_dir}", flush=True)
    logger.info("agent.init repo_root=%s skills_dir=%s", repo_root, skills_dir)

    agent = build_agent()
    config = build_agent_config(thread_id)
    skills_files = get_all_skills(skills_dir)
    log_skills_content(skills_files)
    print(f"skills 已加载: {len(skills_files)} 个", flush=True)
    for skill_path in sorted(skills_files):
        print(f"  - {skill_path}", flush=True)
    print(f"已就绪（thread_id={thread_id}）。q / quit / exit / 退出 结束。", flush=True)

    messages: list[dict[str, str]] = []
    while True:
        try:
            user_input = input("\n用户: ").strip()
        except EOFError:
            print("\n(EOF)", flush=True)
            logger.info("repl.eof")
            break

        if not user_input:
            continue
        if user_input.lower() in ("退出", "quit", "exit", "q"):
            print("再见。", flush=True)
            logger.info("repl.exit")
            break

        logger.info("turn.start user_input=%s", user_input)
        messages.append({"role": "user", "content": user_input})
        result = await invoke_agent(
            agent=agent,
            messages=messages,
            config=config,
        )
        log_tool_events(result)
        assistant_text = extract_text(result)
        messages.append({"role": "assistant", "content": assistant_text})
        logger.info("turn.end assistant_output=%s", assistant_text)
        print(f"\n助手: {assistant_text}", flush=True)

    return 0


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate generated opencli skills interactively")
    parser.add_argument(
        "--thread-id",
        default=os.getenv("SKILLS_VERIFY_THREAD_ID", DEFAULT_THREAD_ID),
        help="checkpoint thread_id",
    )
    return parser.parse_args()


def main() -> int:
    setup_logging()
    load_environment()
    args = parse_args()
    return asyncio.run(run_repl(args.thread_id))


if __name__ == "__main__":
    raise SystemExit(main())
