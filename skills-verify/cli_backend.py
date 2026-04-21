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
        executable: str = "openapi-cli",
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
        self.executable = executable
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

        path_parts = [part for part in self._env.get("PATH", "").split(os.pathsep) if part]
        for candidate in (self.app_dir / "bin", self.app_dir):
            candidate_text = str(candidate)
            if candidate.exists() and candidate_text not in path_parts:
                path_parts.insert(0, candidate_text)
        self._env["PATH"] = os.pathsep.join(path_parts)

    def _parse(self, command: str) -> list[str] | None:
        try:
            args = shlex.split(command)
        except ValueError:
            return None

        if not args:
            return None
        if os.path.basename(args[0]) != self.executable:
            return None
        if any(arg in SHELL_CONTROL_TOKENS for arg in args[1:]):
            return None
        args[0] = self.executable
        return args

    def execute(self, command: str, *, timeout: int | None = None) -> ExecuteResponse:
        args = self._parse(command)
        if args is None:
            return ExecuteResponse(
                output="Error: Only openapi-cli commands are allowed.",
                exit_code=126,
            )

        try:
            result = subprocess.run(
                args,
                check=False,
                capture_output=True,
                text=True,
                timeout=timeout or self.timeout,
                cwd=str(self.cwd),
                env=self._env.copy(),
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
