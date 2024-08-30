{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "helm.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Calculate the leader election based on replicas.
*/}}
{{- define "calculatedValue.leaderElection" -}}
{{- if gt (int .Values.replicaCount) 1}}true{{- else }}false{{- end -}}
{{- end -}}
