import sql_mcp_server


def test_package_has_version() -> None:
    assert sql_mcp_server.__version__ == "0.1.0"
