package main

var runner = `package main

import (
{{- if .Setup.PkgPath }}
	{{ .Setup.PkgName }} "{{ .Setup.PkgPath }}"
{{- end }}
{{- range .Packages }}
	{{ .PkgName }} "{{ .PkgPath }}"
{{- end }}
	e2e{{ .Noise }} "github.com/gombrii/go-e2e"
)

func main() {
	e2e{{ .Noise }}.Runner{
	{{- if .Setup.BeforeRun}}
		BeforeRun: {{ .Setup.PkgName }}.{{ .Setup.BeforeRun }},
	{{- end }}
	{{- if .Setup.AfterRun }}
		AfterRun: {{ .Setup.PkgName }}.{{ .Setup.AfterRun }},
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
