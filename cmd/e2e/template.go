package main

var runner = `package main

import (
{{- if .Setup.PkgPath }}
	{{ .Setup.PkgName }} "{{ .Setup.PkgPath }}"
{{- end }}
{{- range .Packages }}
{{- if ne .PkgPath $.Setup.PkgPath }}
	{{ .PkgName }} "{{ .PkgPath }}"
{{- end }}
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
		Verbose: {{ .Verbose }},
	}.Run(
{{- range .Packages }}
	{{- $pkg := . }}
	{{- range .ExportedVars }}
		{{ $pkg.PkgName }}.{{ .VarName }},
	{{- end }}
{{- end }}
	)
}`
