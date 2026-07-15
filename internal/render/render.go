package render

import (
	"bytes"
	"fmt"
	"go/format"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
)

// templateCache caches parsed templates keyed by template name to avoid
// re-parsing the same template on every renderTemplate call.
var templateCache sync.Map

func writeTemplates(outputDir string, files []generatedFile) error {
	for _, file := range files {
		content, err := renderTemplate(file.Template, file.Data)
		if err != nil {
			return err
		}
		if isGoSourceTemplate(file.Template) {
			formatted, err := format.Source(content)
			if err != nil {
				return fmt.Errorf("format %s: %w", file.Template, err)
			}
			content = formatted
		}
		if err := writeFile(filepath.Join(outputDir, file.Path), content, file.Mode); err != nil {
			return err
		}
	}
	return nil
}

func isGoSourceTemplate(name string) bool {
	return strings.HasPrefix(name, "go/") && strings.HasSuffix(name, ".go.tmpl")
}

func renderTemplate(name string, data any) ([]byte, error) {
	var tmpl *template.Template
	if cached, ok := templateCache.Load(name); ok {
		tmpl = cached.(*template.Template)
	} else {
		raw, err := readTemplate(name)
		if err != nil {
			return nil, err
		}
		parsed, err := template.New(name).Funcs(template.FuncMap{
			"pascal":                          pascal,
			"bodyFlagHelp":                    bodyFlagHelp,
			"cargoPackageName":                cargoPackageName,
			"goType":                          goType,
			"groupHasBodyInput":               groupHasBodyInput,
			"groupHasHeaderParams":            groupHasHeaderParams,
			"groupHasBodyFields":              groupHasBodyFields,
			"groupUsesMCPHTTP":                groupUsesMCPHTTP,
			"groupUsesMCPStdio":               groupUsesMCPStdio,
			"appHasMCPHTTP":                   appHasMCPHTTP,
			"appHasMCPStdio":                  appHasMCPStdio,
			"appHasAnyMCP":                    appHasAnyMCP,
			"appUsesToken":                    appUsesToken,
			"appUsesAKSK":                     appUsesAKSK,
			"appSignerProfile":                appSignerProfile,
			"appSigner":                       appSigner,
			"goString":                        goString,
			"rustString":                      rustString,
			"groupPackageName":                groupPackageName,
			"operationHasHeaderParams":        operationHasHeaderParams,
			"operationHasUserHeaders":         operationHasUserHeaders,
			"operationHasPathParams":          operationHasPathParams,
			"operationHasQueryParams":         operationHasQueryParams,
			"isHiddenAuthHeader":              isHiddenAuthHeader,
			"goAppVersion":                    goAppVersion,
			"rustAppVersion":                  rustAppVersion,
			"cliFlagName":                     cliFlagName,
			"cliParamFlagName":                cliParamFlagName,
			"cliBodyFlagName":                 cliBodyFlagName,
			"rustFieldName":                   rustFieldName,
			"rustTypeName":                    rustTypeName,
			"rustParamFieldName":              rustParamFieldName,
			"rustParamFlagName":               rustParamFlagName,
			"rustBodyFieldName":               rustBodyFieldName,
			"rustBodyFlagName":                rustBodyFlagName,
			"rustBodyFieldsForSigner":         rustBodyFieldsForSigner,
			"rustModuleName":                  rustModuleName,
			"rustType":                        rustType,
			"stringMapLiteral":                stringMapLiteral,
			"stringSliceLiteral":              stringSliceLiteral,
			"exampleValue":                    exampleValue,
			"exampleJSONFields":               exampleJSONFields,
			"bodyRequiredLabel":               bodyRequiredLabel,
			"demoRequestJSON":                 demoRequestJSON,
			"operationIsWriteMethod":          operationIsWriteMethod,
			"operationRiskLabel":              operationRiskLabel,
			"groupDocumentationIssues":        groupDocumentationIssues,
			"appDocumentationIssues":          appDocumentationIssues,
			"groupedGroupDocumentationIssues": groupedGroupDocumentationIssues,
			"groupedAppDocumentationIssues":   groupedAppDocumentationIssues,
			"hasOptionalFields":               hasOptionalFields,
			"upper":                           strings.ToUpper,
		}).Parse(string(raw))
		if err != nil {
			return nil, err
		}
		templateCache.Store(name, parsed)
		tmpl = parsed
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
