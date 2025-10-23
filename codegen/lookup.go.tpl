package ticketmaster

import (
    "fmt"
)

{{- range .Lookups }}
{{ $lookup := . }}
// {{ .Options.Description}}
type {{ .Name }} {{ .GoType }}

const (
{{- range .Entries }}
    {{ .GoLabel }} {{ $lookup.Name }} = {{ .Id.Value }}
{{- end }}
)

type {{ .Name }}Lookup struct {
{{- range .Fields }}
    {{ .Name }} {{ .GoType }}
{{- end }}
}

func (x {{ .Name }}) Lookup() {{ .Name }}Lookup {
    switch  x {
{{- range .Entries }}
    case {{ .GoLabel }}:
        return {{ $lookup.Name }}Lookup{
{{- range .Values }}
            {{ .Field.Name }}: {{ .GoValue }},
{{- end }}
        }   
{{- end }}
    }
    panic(fmt.Sprintf("unknown {{ .Name }} value %v", x))
}

{{- end }}
