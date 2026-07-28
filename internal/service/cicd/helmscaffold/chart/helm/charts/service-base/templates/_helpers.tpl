{{- define "service-base.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "service-base.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- define "service-base.labels" -}}
app.kubernetes.io/name: {{ include "service-base.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
{{- define "service-base.selectorLabels" -}}
app.kubernetes.io/name: {{ .Values.selectorName | default (include "service-base.name" .) }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
