package codegen

import (
	"bytes"
	"go/format"
	"strings"
	"testing"
	"text/template"

	"github.com/freeconf/yang/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const containerTestExtensions = `
	prefix "x";
	extension collection {
		argument "name";
	}
	extension scopes {
		argument "value";
	}
	extension col {
		argument "name";
	}
	extension struct {
		argument "name";
	}
	extension nullable {
		description "may be null";
	}
`

// loadContainerModule wraps body in a module that declares the extensions and
// reads every list in names as a crud entry.
func loadContainerModule(t *testing.T, body string, names ...string) (*Cruder, error) {
	t.Helper()
	m, err := parser.LoadModuleFromString(nil, "module x {"+containerTestExtensions+body+"}")
	require.NoError(t, err)
	opts := CrudOptions{}
	for _, n := range names {
		opts.Entries = append(opts.Entries, CrudOptionsEntry{Table: n, Ydef: n})
	}
	c := NewCruder(opts)
	return c, c.read(m)
}

// reportYang mirrors the shape of an RFC 6035 voice-quality document: a key at
// the top level and metrics nested two levels down.
const reportYang = `
	list report {
		x:collection "reports";
		key id;
		leaf id {
			type string;
			x:col "_id";
		}
		container publish {
			description "Envelope the report arrived in.";
			x:scopes "read";
			leaf from {
				type string;
			}
		}
		container rfc6035 {
			x:scopes "read";
			leaf id {
				description "Shares the list key's name but is not a key.";
				type string;
				x:col "CallID";
			}
			container qualityEst {
				x:col "QualityEst";
				x:scopes "read";
				leaf mosLq {
					type decimal64;
					x:col "MOSLQ";
					x:nullable;
				}
			}
			container delay {
				x:col "Delay";
				x:scopes "read";
				x:struct "ReportDelay";
				leaf rtd {
					type int32;
					x:col "RTD";
				}
			}
		}
		leaf processedAt {
			type string;
			x:col "processed_at";
		}
	}
`

func TestContainers(t *testing.T) {
	c, err := loadContainerModule(t, reportYang, "report")
	require.NoError(t, err)
	r := c.Entries[0]

	require.Len(t, r.fields, 2, "only leaves directly in the list are list fields")
	assert.Equal(t, "Id", r.fields[0].Name())
	assert.True(t, r.fields[0].IsKey())
	assert.Equal(t, "ProcessedAt", r.fields[1].Name())

	require.Len(t, r.Containers(), 2)
	pub := r.Containers()[0]
	assert.Equal(t, "Publish", pub.Name())
	assert.Equal(t, "*ReportPublish", pub.GoType())
	assert.Equal(t, "publish,omitempty", pub.JsonTags())
	assert.Equal(t, "publish,omitempty", pub.BsonTags())
	assert.Equal(t, ` jsonschema:"Envelope the report arrived in."`, pub.JsonSchemaTag())

	var names []string
	for _, n := range r.NestedStructs() {
		names = append(names, n.Struct())
	}
	assert.Equal(t, []string{"ReportPublish", "ReportRfc6035", "ReportRfc6035QualityEst", "ReportDelay"}, names,
		"depth first, parent name plus container name unless x:struct overrides it")

	rfc := r.NestedStructs()[1]
	callID := rfc.Fields()[0]
	assert.False(t, callID.IsKey(), "a nested leaf named like the list key must not be treated as the key")
	assert.Equal(t, "CallID", callID.BsonTags("read"))

	qe := r.NestedStructs()[2]
	assert.Equal(t, "QualityEst", qe.Name())
	assert.Equal(t, "qualityEst,omitempty", qe.JsonTags())
	assert.Equal(t, "QualityEst,omitempty", qe.BsonTags(), "x:col sets the stored document key")

	mos := qe.Fields()[0]
	assert.Equal(t, "*float64", mos.GoType())
	assert.Equal(t, "mosLq,omitempty", mos.JsonTags("read"))
	assert.Equal(t, "MOSLQ,omitempty", mos.BsonTags("read"))
}

func TestContainerRules(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name: "needs a Mongo collection",
			body: `list t {
				container c {
					x:scopes "read";
					leaf l { type string; }
				}
			}`,
			wantErr: "x:collection",
		},
		{
			name: "default scopes are not read-only",
			body: `list t {
				x:collection "t";
				container c {
					leaf l { type string; }
				}
			}`,
			wantErr: "not supported yet",
		},
		{
			name: "create scope is rejected",
			body: `list t {
				x:collection "t";
				container c {
					x:scopes "read,create";
					leaf l { type string; }
				}
			}`,
			wantErr: "not supported yet",
		},
		{
			name: "list inside a container",
			body: `list t {
				x:collection "t";
				container c {
					x:scopes "read";
					list items {
						leaf l { type string; }
					}
				}
			}`,
			wantErr: "not supported inside a container",
		},
		{
			name: "list inside a list",
			body: `list t {
				list items {
					leaf l { type string; }
				}
			}`,
			wantErr: "not supported",
		},
		{
			name: "struct name collides with a list",
			body: `list t {
				x:collection "t";
				container c {
					x:scopes "read";
					x:struct "T";
					leaf l { type string; }
				}
			}`,
			wantErr: "already used",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := loadContainerModule(t, test.body, "t")
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
		})
	}
}

// TestContainerTemplate renders the struct section the way a service template
// would and checks the result is valid, gofmt-able Go.
func TestContainerTemplate(t *testing.T) {
	c, err := loadContainerModule(t, reportYang, "report")
	require.NoError(t, err)

	tmpl := template.Must(template.New("t").Parse(`package x
{{- range .Entries }}
type {{ .Struct }} struct {
{{- range .Fields }}
	{{ .Name }} {{ .GoType }} ` + "`" + `json:"{{ .JsonTags "read" }}" bson:"{{ .BsonTags "read" }}"` + "`" + `
{{- end }}
{{- range .Containers }}
	{{ .Name }} {{ .GoType }} ` + "`" + `json:"{{ .JsonTags }}" bson:"{{ .BsonTags }}"{{ .JsonSchemaTag }}` + "`" + `
{{- end }}
}
{{- range .NestedStructs }}
type {{ .Struct }} struct {
{{- range .Fields }}
	{{ .Name }} {{ .GoType }} ` + "`" + `json:"{{ .JsonTags "read" }}" bson:"{{ .BsonTags "read" }}"` + "`" + `
{{- end }}
{{- range .Containers }}
	{{ .Name }} {{ .GoType }} ` + "`" + `json:"{{ .JsonTags }}" bson:"{{ .BsonTags }}"{{ .JsonSchemaTag }}` + "`" + `
{{- end }}
}
{{- end }}
{{- end }}
`))
	var out bytes.Buffer
	require.NoError(t, tmpl.Execute(&out, c))
	src, err := format.Source(out.Bytes())
	require.NoError(t, err, "generated code should be valid Go:\n%s", out.String())

	// gofmt aligns struct fields into columns, collapse that padding before matching
	gen := strings.Join(strings.Fields(string(src)), " ")
	for _, want := range []string{
		"Rfc6035 *ReportRfc6035 `json:\"rfc6035,omitempty\" bson:\"rfc6035,omitempty\"`",
		"QualityEst *ReportRfc6035QualityEst `json:\"qualityEst,omitempty\" bson:\"QualityEst,omitempty\"`",
		"MosLq *float64 `json:\"mosLq,omitempty\" bson:\"MOSLQ,omitempty\"`",
		"type ReportDelay struct",
	} {
		assert.True(t, strings.Contains(gen, want), "missing %q in:\n%s", want, gen)
	}
}
