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


def resolve_repo_root() -> Path:
    return Path(__file__).resolve().parents[1]


def resolve_model():
    model_path = os.environ.get("SKILLS_VERIFY_MODEL")
    if not model_path:
        raise SystemExit(
            "Set SKILLS_VERIFY_MODEL to a Python import path such as package.module:model."
        )
    module_name, attr_name = model_path.split(":", 1)
    module = __import__(module_name, fromlist=[attr_name])
    return getattr(module, attr_name)


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
        model=resolve_model(),
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
    args = parse_args()
    prompt = args.prompt.strip()
    if not prompt:
        raise SystemExit("Prompt must not be empty.")
    print(asyncio.run(run_prompt(prompt)))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
