{{/*
Expand the name of the chart.
*/}}
{{- define "databasus.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "databasus.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "databasus.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "databasus.labels" -}}
helm.sh/chart: {{ include "databasus.chart" . }}
{{ include "databasus.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "databasus.selectorLabels" -}}
app.kubernetes.io/name: {{ include "databasus.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app: databasus
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "databasus.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "databasus.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Namespace - uses the release namespace from helm install -n <namespace>
*/}}
{{- define "databasus.namespace" -}}
{{- .Release.Namespace }}
{{- end }}
