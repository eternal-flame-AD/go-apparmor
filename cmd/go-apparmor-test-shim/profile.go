package main

import "text/template"

type profileTplContext struct {
	ExecPath       string
	ExecDotPath    string
	Flags          string
	ProfileName    string
	SubProfileName string
	HatName        string
}

var profileTpl = template.Must(template.New("").Parse(`

include <tunables/global>

profile {{ .ProfileName }} {{.ExecPath}} flags=({{ .Flags }}) {
	include <abstractions/base>
	include <abstractions/apparmor_api/introspect>
	
	{{.ExecPath}} mr,

	include if exists <local/{{.ExecDotPath}}>

	^{{ .HatName }} {
		include <abstractions/base>
		include <abstractions/apparmor_api/introspect>
	
		{{.ExecPath}} mr,	
	}

	change_profile -> {{ .SubProfileName }},

	profile {{.SubProfileName}} {
		include <abstractions/base>
		include <abstractions/apparmor_api/introspect>

		{{.ExecPath}} mr,
	}
}
`))
