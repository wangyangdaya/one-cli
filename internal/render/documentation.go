package render

import (
	"maps"
	"slices"
	"strings"

	"one-cli/internal/model"
)

func groupDocumentationIssues(group model.Group, lang string) []string {
	zh := strings.EqualFold(strings.TrimSpace(lang), "zh")
	issue := func(en, cn string) string {
		if zh {
			return cn
		}
		return en
	}

	var issues []string
	if strings.TrimSpace(group.Description) == "" {
		issues = append(issues, issue(
			"`"+group.Name+"` is missing a group description",
			"`"+group.Name+"` 缺少分组描述",
		))
	}
	for _, op := range group.Operations {
		if strings.TrimSpace(op.Summary) == "" {
			issues = append(issues, issue(
				"`"+op.CommandName+"` is missing a command summary",
				"`"+op.CommandName+"` 缺少命令摘要",
			))
		}
		issues = append(issues, operationParameterIssues(op, issue)...)
		issues = append(issues, operationBodyFieldIssues(op, issue)...)
		issues = append(issues, operationPathIssues(op, issue)...)
	}
	return issues
}

func appDocumentationIssues(app model.App, lang string) []string {
	var issues []string
	for _, group := range app.Groups {
		issues = append(issues, groupDocumentationIssues(group, lang)...)
	}
	return issues
}

type documentationIssueGroup struct {
	Title string
	Items []string
}

func groupedAppDocumentationIssues(app model.App, lang string) []documentationIssueGroup {
	var groups []documentationIssueGroup
	for _, group := range app.Groups {
		groups = append(groups, groupedGroupDocumentationIssues(group, lang)...)
	}
	return groups
}

func groupedGroupDocumentationIssues(group model.Group, lang string) []documentationIssueGroup {
	zh := strings.EqualFold(strings.TrimSpace(lang), "zh")
	text := func(en, cn string) string {
		if zh {
			return cn
		}
		return en
	}

	var groups []documentationIssueGroup
	if strings.TrimSpace(group.Description) == "" {
		groups = append(groups, documentationIssueGroup{
			Title: text("`"+group.Name+"` is missing a group description", "`"+group.Name+"` 缺少分组描述"),
		})
	}
	for _, op := range group.Operations {
		items := operationDocumentationIssueItems(op, text)
		if len(items) == 0 {
			continue
		}
		groups = append(groups, documentationIssueGroup{
			Title: "`" + op.CommandName + "`",
			Items: items,
		})
	}
	return groups
}

func operationDocumentationIssueItems(op model.Operation, text func(string, string) string) []string {
	var items []string
	if strings.TrimSpace(op.Summary) == "" {
		items = append(items, text("command summary: missing", "命令摘要：缺少"))
	}
	items = append(items, operationParameterIssueItems(op, text)...)
	items = append(items, operationBodyFieldIssueItems(op, text)...)
	items = append(items, operationPathIssueItems(op, text)...)
	return items
}

func operationParameterIssueItems(op model.Operation, text func(string, string) string) []string {
	var items []string
	seen := make(map[string]bool, len(op.Parameters))
	for _, param := range op.Parameters {
		name := strings.TrimSpace(param.Name)
		location := strings.TrimSpace(param.In)
		if name == "" {
			continue
		}
		var problems []string
		if location == "" {
			problems = append(problems, text("missing location", "缺少位置"))
		} else if !supportedParameterLocation(location) {
			problems = append(problems, text("unsupported location `"+location+"`", "不支持的位置 `"+location+"`"))
		}
		key := location + "\x00" + name
		if seen[key] {
			problems = append(problems, text("duplicate declaration", "重复声明"))
		}
		seen[key] = true
		if strings.TrimSpace(param.Description) == "" {
			problems = append(problems, text("missing description", "缺少说明"))
		}
		if strings.TrimSpace(param.Type) == "" {
			problems = append(problems, text("missing type", "缺少类型"))
		}
		if len(problems) > 0 {
			items = append(items, text("parameter `"+name+"`: ", "参数 `"+name+"`：")+strings.Join(problems, "; "))
		}
	}
	return items
}

func operationBodyFieldIssueItems(op model.Operation, text func(string, string) string) []string {
	var items []string
	seen := make(map[string]bool, len(op.BodyFields)+len(op.BodySchemaFields))
	for _, field := range append(append([]model.BodyField(nil), op.BodySchemaFields...), op.BodyFields...) {
		name := strings.TrimSpace(field.Name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		var problems []string
		if strings.TrimSpace(field.Description) == "" {
			problems = append(problems, text("missing description", "缺少说明"))
		}
		if strings.TrimSpace(field.Type) == "" {
			problems = append(problems, text("missing type", "缺少类型"))
		}
		if field.RequiredUnknown {
			problems = append(problems, text("required is unspecified", "未声明是否必填"))
		}
		if len(problems) > 0 {
			items = append(items, text("request body field `"+name+"`: ", "请求体字段 `"+name+"`：")+strings.Join(problems, "; "))
		}
	}
	return items
}

func operationPathIssueItems(op model.Operation, text func(string, string) string) []string {
	path := strings.TrimSpace(op.Path)
	templateParams := pathTemplateParams(path)
	declaredPathParams := make(map[string]model.Parameter, len(op.Parameters))
	pathParamProblems := make(map[string][]string)
	for _, param := range op.Parameters {
		if strings.TrimSpace(param.In) != "path" {
			continue
		}
		name := strings.TrimSpace(param.Name)
		if name == "" {
			continue
		}
		declaredPathParams[name] = param
		if !templateParams[name] {
			pathParamProblems[name] = append(pathParamProblems[name], text("not present in path `"+path+"`", "路径 `"+path+"` 中不存在该占位符"))
		}
		if !param.Required {
			pathParamProblems[name] = append(pathParamProblems[name], text("should be required", "应声明为必填"))
		}
	}
	for _, name := range slices.Sorted(maps.Keys(templateParams)) {
		if _, ok := declaredPathParams[name]; !ok {
			pathParamProblems["{"+name+"}"] = append(pathParamProblems["{"+name+"}"], text("missing matching `in: path` parameter", "缺少匹配的 `in: path` 参数声明"))
		}
	}
	var items []string
	for _, name := range slices.Sorted(maps.Keys(pathParamProblems)) {
		items = append(items, text("path parameter `"+name+"`: ", "路径参数 `"+name+"`：")+strings.Join(pathParamProblems[name], "; "))
	}
	return items
}

func operationParameterIssues(op model.Operation, issue func(string, string) string) []string {
	var issues []string
	seen := make(map[string]bool, len(op.Parameters))
	for _, param := range op.Parameters {
		name := strings.TrimSpace(param.Name)
		location := strings.TrimSpace(param.In)
		if name == "" {
			continue
		}
		if location == "" {
			issues = append(issues, issue(
				"`"+op.CommandName+"` parameter `"+name+"` is missing a location",
				"`"+op.CommandName+"` 的参数 `"+name+"` 缺少位置",
			))
		} else if !supportedParameterLocation(location) {
			issues = append(issues, issue(
				"`"+op.CommandName+"` parameter `"+name+"` uses unsupported location `"+location+"`",
				"`"+op.CommandName+"` 的参数 `"+name+"` 使用了不支持的位置 `"+location+"`",
			))
		}
		key := location + "\x00" + name
		if seen[key] {
			issues = append(issues, issue(
				"`"+op.CommandName+"` declares duplicate `"+location+"` parameter `"+name+"`",
				"`"+op.CommandName+"` 重复声明了 `"+location+"` 参数 `"+name+"`",
			))
		}
		seen[key] = true
		if strings.TrimSpace(param.Description) == "" {
			issues = append(issues, issue(
				"`"+op.CommandName+"` parameter `"+name+"` is missing a description",
				"`"+op.CommandName+"` 的参数 `"+name+"` 缺少说明",
			))
		}
		if strings.TrimSpace(param.Type) == "" {
			issues = append(issues, issue(
				"`"+op.CommandName+"` parameter `"+name+"` is missing a type",
				"`"+op.CommandName+"` 的参数 `"+name+"` 缺少类型",
			))
		}
	}
	return issues
}

func operationBodyFieldIssues(op model.Operation, issue func(string, string) string) []string {
	var issues []string
	seen := make(map[string]bool, len(op.BodyFields)+len(op.BodySchemaFields))
	for _, field := range append(append([]model.BodyField(nil), op.BodySchemaFields...), op.BodyFields...) {
		name := strings.TrimSpace(field.Name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		if strings.TrimSpace(field.Description) == "" {
			issues = append(issues, issue(
				"`"+op.CommandName+"` request body field `"+name+"` is missing a description",
				"`"+op.CommandName+"` 的请求体字段 `"+name+"` 缺少说明",
			))
		}
		if strings.TrimSpace(field.Type) == "" {
			issues = append(issues, issue(
				"`"+op.CommandName+"` request body field `"+name+"` is missing a type",
				"`"+op.CommandName+"` 的请求体字段 `"+name+"` 缺少类型",
			))
		}
		if field.RequiredUnknown {
			issues = append(issues, issue(
				"`"+op.CommandName+"` request body field `"+name+"` does not declare whether it is required",
				"`"+op.CommandName+"` 的请求体字段 `"+name+"` 未声明是否必填",
			))
		}
	}
	return issues
}

func operationPathIssues(op model.Operation, issue func(string, string) string) []string {
	var issues []string
	path := strings.TrimSpace(op.Path)
	templateParams := pathTemplateParams(path)
	declaredPathParams := make(map[string]model.Parameter, len(op.Parameters))
	for _, param := range op.Parameters {
		if strings.TrimSpace(param.In) != "path" {
			continue
		}
		name := strings.TrimSpace(param.Name)
		if name == "" {
			continue
		}
		declaredPathParams[name] = param
		if !templateParams[name] {
			issues = append(issues, issue(
				"`"+op.CommandName+"` declares path parameter `"+name+"` that is not present in path `"+path+"`",
				"`"+op.CommandName+"` 声明了路径参数 `"+name+"`，但路径 `"+path+"` 中不存在该占位符",
			))
		}
		if !param.Required {
			issues = append(issues, issue(
				"`"+op.CommandName+"` path parameter `"+name+"` should be required",
				"`"+op.CommandName+"` 的路径参数 `"+name+"` 应声明为必填",
			))
		}
	}
	for _, name := range slices.Sorted(maps.Keys(templateParams)) {
		if _, ok := declaredPathParams[name]; !ok {
			issues = append(issues, issue(
				"`"+op.CommandName+"` path parameter `{"+name+"}` is missing a matching `in: path` parameter",
				"`"+op.CommandName+"` 的路径参数 `{"+name+"}` 缺少匹配的 `in: path` 参数声明",
			))
		}
	}
	return issues
}

func supportedParameterLocation(location string) bool {
	switch strings.TrimSpace(location) {
	case "path", "query", "header":
		return true
	default:
		return false
	}
}

func pathTemplateParams(path string) map[string]bool {
	params := make(map[string]bool)
	for {
		start := strings.Index(path, "{")
		if start < 0 {
			return params
		}
		path = path[start+1:]
		end := strings.Index(path, "}")
		if end < 0 {
			return params
		}
		if name := strings.TrimSpace(path[:end]); name != "" {
			params[name] = true
		}
		path = path[end+1:]
	}
}
