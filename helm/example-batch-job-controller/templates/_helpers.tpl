{{/* vim: set filetype=mustache: */}}

{{/* Create custom SIX values */}}
{{- define "batch-job-controller.name" -}}
{{- printf "%s" .Values.name | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}


{{/* app labels */}}
{{- define "batch-job-controller.app-labels" -}}
app: {{ .Values.name }}
{{- end -}}

{{/* helm labels */}}
{{- define "batch-job-controller.helm-labels" -}}
chart: {{ .Chart.Name }}-{{ .Chart.Version }}
component: {{ .Values.name }}
heritage: {{ .Release.Service }}
release: {{ .Release.Name }}
{{- end -}}

{{/* monitoring labels / annotations */}}
{{- define "batch-job-controller.monitoring" -}}
{{- if .Values.monitoring -}}
{{ toYaml .Values.monitoring | trim }}
{{- end -}}
{{- end }}

{{/* deployment labels */}}
{{- define "batch-job-controller.deployment-labels" -}}
{{- if .Values.deployment.labels -}}
{{ toYaml .Values.deployment.labels | trim }}
{{- end -}}
{{- end }}
