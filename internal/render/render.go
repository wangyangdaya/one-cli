package render

import (
	"bytes"
	"fmt"
	"go/format"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"one-cli/internal/runtimeconfig"
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
			"goParamFieldName":                goParamFieldName,
			"goParamFlagName":                 goParamFlagName,
			"goBodyFieldName":                 goBodyFieldName,
			"goBodyFieldHasFlag":              goBodyFieldHasFlag,
			"goBodyFlagName":                  goBodyFlagName,
			"goInputFieldName":                goInputFieldName,
			"groupHasBodyInput":               groupHasBodyInput,
			"groupHasHeaderParams":            groupHasHeaderParams,
			"groupHasBodyFields":              groupHasBodyFields,
			"groupHasDataBody":                groupHasDataBody,
			"groupUsesMCPHTTP":                groupUsesMCPHTTP,
			"groupUsesMCPStdio":               groupUsesMCPStdio,
			"appHasMCPHTTP":                   appHasMCPHTTP,
			"appHasMCPStdio":                  appHasMCPStdio,
			"appHasAnyMCP":                    appHasAnyMCP,
			"appUsesToken":                    appUsesToken,
			"appUsesAPIKey":                   appUsesAPIKey,
			"appUsesAKSK":                     appUsesAKSK,
			"appSignerProfile":                appSignerProfile,
			"appSigner":                       appSigner,
			"goString":                        goString,
			"rustString":                      rustString,
			"groupPackageName":                groupPackageName,
			"skillName":                       skillName,
			"operationHasHeaderParams":        operationHasHeaderParams,
			"operationHasUserHeaders":         operationHasUserHeaders,
			"groupHasUserHeaders":             groupHasUserHeaders,
			"operationHasPathParams":          operationHasPathParams,
			"operationHasQueryParams":         operationHasQueryParams,
			"groupHasFileFields":              groupHasFileFields,
			"defaultFileField":                defaultFileField,
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
			"rustBodyFieldHasFlag":            rustBodyFieldHasFlag,
			"rustBodyFieldsForSigner":         rustBodyFieldsForSigner,
			"rustModuleName":                  rustModuleName,
			"rustType":                        rustType,
			"stringMapLiteral":                stringMapLiteral,
			"stringSliceLiteral":              stringSliceLiteral,
			"exampleValue":                    exampleValue,
			"bodyFieldExample":                bodyFieldExample,
			"parameterExample":                parameterExample,
			"flattenBodyFields":               flattenBodyFields,
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
			"runtimeEnvelopePrefix":           func() string { return runtimeconfig.EnvelopePrefix },
			"runtimeEnvelopeSuffix":           func() string { return runtimeconfig.EnvelopeSuffix },
			"runtimeAADPrefix":                func() string { return runtimeconfig.AADPrefix },
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
