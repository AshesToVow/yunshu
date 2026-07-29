{{- define "deployment-base.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "deployment-base.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- define "deployment-base.labels" -}}
app.kubernetes.io/name: {{ include "deployment-base.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}
{{- define "deployment-base.selectorLabels" -}}
app.kubernetes.io/name: {{ include "deployment-base.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
