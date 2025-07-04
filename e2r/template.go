package main

var runner = `package main

import (
{{- if .Hooks.PkgPath }}
	{{ .Hooks.PkgName }} "{{ .Hooks.PkgPath }}"
{{- end }}
{{- range .Packages }}
	{{ .PkgName }} "{{ .PkgPath }}"
{{- end }}
	"github.com/gombrii/go-e2e"
)

func main() {
	e2e.Runner{
	{{- if .Hooks.BeforeRun}}
		BeforeRun: {{ .Hooks.PkgName }}.{{ .Hooks.BeforeRun }},
	{{- end }}
	{{- if .Hooks.AfterRun }}
		AfterRun: {{ .Hooks.PkgName }}.{{ .Hooks.AfterRun }},
	{{- end }}
	}.Run(
{{- range .Packages }}
	{{- $pkg := . }}
	{{- range .ExportedVars }}
		{{ $pkg.PkgName }}.{{ .VarName }},
	{{- end }}
{{- end }}
	)
}`
