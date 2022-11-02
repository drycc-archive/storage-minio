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

{{- define "tikv.addresses" -}}
{{- if eq .Values.global.storageLocation "on-cluster" }}
{{- $replicaCount := int .Values.meta.tikv.replicas }}
{{- $clusterDomain := .Values.global.clusterDomain }}
{{- $messages := list -}}
{{ range $i := until $replicaCount }}
  {{- $messages = printf "http://drycc-storage-meta-tikv-%d.drycc-storage-meta-tikv.%s.svc.%s:20180" $i $.Release.Namespace $clusterDomain | append $messages -}}
{{ end }}
{{- $message := join "," $messages -}}
{{- printf "%s" $message }}
{{- else }}
{{- printf "" }}
{{- end -}}
{{- end -}}
