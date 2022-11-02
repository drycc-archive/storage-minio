{{- define "pd.addresses" -}}
{{- if eq .Values.global.storageLocation "on-cluster" }}
{{- $replicaCount := int .Values.meta.pd.replicas }}
{{- $clusterDomain := .Values.global.clusterDomain }}
{{- $messages := list -}}
{{ range $i := until $replicaCount }}
  {{- $messages = printf "http://drycc-storage-meta-pd-%d.drycc-storage-meta-pd.%s.svc.%s:2379" $i $.Release.Namespace $clusterDomain | append $messages -}}
{{ end }}
{{- $message := join "," $messages -}}
{{- printf "%s" $message }}
{{- else }}
{{- printf "" }}
{{- end -}}
{{- end -}}
