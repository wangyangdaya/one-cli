from __future__ import annotations

import argparse
import asyncio
import logging
import os
from pathlib import Path

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

    return create_deep_agent(
        model=build_llm(),
        backend=CliBackend(repo_root=repo_root, app_dir=app_dir),
        system_prompt=SYSTEM_PROMPT,
        skills=[str(skills_dir)],
        checkpointer=MemorySaver(),
        name="openapi_skills_verifier",
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


async def invoke_agent(
    *,
    agent,
    messages: list[dict[str, str]],
    config: dict,
):
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
