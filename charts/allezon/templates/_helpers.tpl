

{{- define "redpanda.fullname" -}}

{{- printf "%s-%s" .Release.Name "redpanda" | trunc 63 | trimSuffix "-" -}}

{{- end }}
