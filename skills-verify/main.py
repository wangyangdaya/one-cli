from __future__ import annotations

import argparse
import asyncio
import os
from pathlib import Path

from clibackend import CliBackend


SYSTEM_PROMPT = """You are validating generated opencli skills.
Always use the provided skills when they apply.
Only use commands that can be executed through the provided CLI backend.
Do not invent APIs or shell commands.
"""

DEFAULT_MODEL_NAME = "gpt-4o-mini"


def resolve_repo_root() -> Path:
    return Path(__file__).resolve().parents[1]


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

    repo_root = resolve_repo_root()
    app_dir = repo_root / "tmp" / "openapi"
    skills_dir = app_dir / "skills"
    if not skills_dir.exists():
        raise SystemExit(f"Skills directory not found: {skills_dir}")

    return create_deep_agent(
        model=build_llm(),
        backend=CliBackend(repo_root=repo_root),
        system_prompt=SYSTEM_PROMPT,
        skills=[str(skills_dir)],
        name="openapi_skills_verifier",
    )


async def run_prompt(prompt: str) -> str:
    agent = build_agent()
    result = await agent.ainvoke({"messages": [{"role": "user", "content": prompt}]})
    if isinstance(result, dict):
        messages = result.get("messages", [])
        if messages:
            last = messages[-1]
            content = getattr(last, "content", last)
            return str(content)
    return str(result)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate generated opencli skills")
    parser.add_argument("--prompt", required=True, help="Prompt to send to the agent")
    return parser.parse_args()


def main() -> int:
    load_environment()
    args = parse_args()
    prompt = args.prompt.strip()
    if not prompt:
        raise SystemExit("Prompt must not be empty.")
    print(asyncio.run(run_prompt(prompt)))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
