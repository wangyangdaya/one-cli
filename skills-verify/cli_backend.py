from __future__ import annotations

import os
import shlex
import subprocess
from pathlib import Path

from deepagents.backends.filesystem import FilesystemBackend
from deepagents.backends.protocol import ExecuteResponse, SandboxBackendProtocol


SHELL_CONTROL_TOKENS = {";", "&&", "||", "|", ">", ">>", "<", "2>", "2>>", "&"}


class CliBackend(FilesystemBackend, SandboxBackendProtocol):
    def __init__(
        self,
        repo_root: str | Path,
        *,
        app_dir: str | Path = "tmp/openapi",
        executables: str | list[str] = "openapi-cli",
        timeout: int = 120,
        env: dict[str, str] | None = None,
        inherit_env: bool = True,
    ) -> None:
        super().__init__(
            root_dir=repo_root,
            virtual_mode=False,
            max_file_size_mb=10,
        )
        self.repo_root = Path(repo_root).resolve()
        resolved_app_dir = Path(app_dir)
        if not resolved_app_dir.is_absolute():
            resolved_app_dir = self.repo_root / resolved_app_dir
        self.app_dir = resolved_app_dir.resolve()
        self.skills_dir = self.app_dir / "skills"
        
        # Support multiple executables
        if isinstance(executables, str):
            # Support comma-separated list from env var
            self.executables = [e.strip() for e in executables.split(",") if e.strip()]
        else:
            self.executables = list(executables)
        
        if not self.executables:
            self.executables = ["openapi-cli"]
        
        self.timeout = timeout
        self.cwd = self.app_dir

        if inherit_env:
            self._env = os.environ.copy()
            if env:
                self._env.update(env)
        else:
            self._env = dict(env or {})

        opencli_base_url = os.getenv("OPENCLI_BASE_URL")
        if opencli_base_url:
            self._env["OPENCLI_BASE_URL"] = opencli_base_url
        self._opencli_base_url_map = self._parse_opencli_base_url_map(
            os.getenv("OPENCLI_BASE_URL_MAP", "")
        )

        path_parts = [part for part in self._env.get("PATH", "").split(os.pathsep) if part]
        for candidate in (self.app_dir / "bin", self.app_dir):
            candidate_text = str(candidate)
            if candidate.exists() and candidate_text not in path_parts:
                path_parts.insert(0, candidate_text)
        self._env["PATH"] = os.pathsep.join(path_parts)

    @staticmethod
    def _parse_opencli_base_url_map(raw_value: str) -> dict[str, str]:
        mappings: dict[str, str] = {}
        for item in raw_value.split(","):
            candidate = item.strip()
            if not candidate or "=" not in candidate:
                continue
            executable, base_url = candidate.split("=", 1)
            executable = executable.strip()
            base_url = base_url.strip()
            if executable and base_url:
                mappings[executable] = base_url
        return mappings

    def _parse(self, command: str) -> list[str] | None:
        try:
            args = shlex.split(command)
        except ValueError:
            return None

        if not args:
            return None
        
        # Check if the command matches any of the allowed executables
        command_basename = os.path.basename(args[0])
        if command_basename not in self.executables:
            return None
        
        if any(arg in SHELL_CONTROL_TOKENS for arg in args[1:]):
            return None
        
        # Keep the original command name (don't replace it)
        return args

    def execute(self, command: str, *, timeout: int | None = None) -> ExecuteResponse:
        args = self._parse(command)
        if args is None:
            allowed = ", ".join(self.executables)
            return ExecuteResponse(
                output=f"Error: Only these commands are allowed: {allowed}",
                exit_code=126,
            )

        command_basename = os.path.basename(args[0])
        env = self._env.copy()
        mapped_base_url = self._opencli_base_url_map.get(command_basename)
        if mapped_base_url:
            env["OPENCLI_BASE_URL"] = mapped_base_url

        try:
            result = subprocess.run(
                args,
                check=False,
                capture_output=True,
                text=True,
                timeout=timeout or self.timeout,
                cwd=str(self.cwd),
                env=env,
            )
        except subprocess.TimeoutExpired:
            return ExecuteResponse(
                output=f"Error: Command timed out after {timeout or self.timeout} seconds.",
                exit_code=124,
            )
        except Exception as exc:  # noqa: BLE001
            return ExecuteResponse(
                output=f"Error executing command ({type(exc).__name__}): {exc}",
                exit_code=1,
            )

        output_parts = []
        if result.stdout:
            output_parts.append(result.stdout.strip())
        if result.stderr:
            output_parts.append(result.stderr.strip())
        output = "\n".join(part for part in output_parts if part) or "<no output>"
        return ExecuteResponse(output=output, exit_code=result.returncode)
