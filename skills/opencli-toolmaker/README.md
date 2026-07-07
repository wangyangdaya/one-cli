# opencli-toolmaker Skill 包

本 Skill 用于教会 Agent 如何使用 `opencli` 工具，把业务方提供的 OpenAPI/Swagger 或 MCP 文档转换为可运行的 CLI 项目和配套 Skill 包。

## 适用场景

- 业务方给了一份 Swagger/OpenAPI JSON/YAML，希望生成 CLI。
- 生成的 CLI 命令名过长、tag 是中文、无法编译。
- 需要把生成的 CLI 再包装成 Agent 可用的 Skill。

## 目录结构

```text
skills/opencli-toolmaker/
├── SKILL.md                           # Agent 入口
├── README.md                          # 本文件
├── assets/
│   ├── opencli.yaml.template          # 配置模板
│   ├── tag-alias.example.yaml         # tag 别名示例
│   └── operation-alias.example.yaml   # operation/path 别名示例
└── references/
    ├── diagnosis-checklist.md         # 输入文档诊断清单
    ├── cleaning-patterns.md           # 清洗与修复模式
    ├── naming-guide.md                # 命名规范
    ├── generation-workflow.md         # 完整生成工作流
    └── troubleshooting.md             # 问题排查
```

## 使用方式

当用户希望基于某份 OpenAPI/MCP 文档生成 CLI 时，Agent 应：

1. 加载本 Skill。
2. 按 `SKILL.md` 中的工作流执行。
3. 遇到具体问题时查阅 `references/` 下对应文档。

## 维护说明

- `SKILL.md` 是 Agent 主入口，保持简洁。
- `references/` 下是详细参考，按需扩展。
- `assets/` 下是可复用的配置模板，可直接复制给用户修改。
