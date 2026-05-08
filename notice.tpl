NOTICE - Third Party Dependencies

This file lists the third-party dependencies used by meta1v and their licenses.

{{ range . -}}
--------------------------------------------------------------------------------
Package: {{ .Name }}
{{ if .Version -}}
Version: {{ .Version }}
{{ end -}}
License: {{ .LicenseName }}
License URL: {{ .LicenseURL }}

{{ end }}
