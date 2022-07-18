{{- define "pd.addresses" -}}
{{- $replicaCount := int .Values.meta.pd.replicas }}
{{- $clusterDomain := .Values.global.clusterDomain }}
{{- $messages := list -}}
{{ range $i := until $replicaCount }}
  {{- $messages = printf "http://drycc-storage-meta-pd-%d.drycc-storage-meta-pd.$(NAMESPACE).svc.%s:2379" $i $clusterDomain | append $messages -}}
{{ end }}
{{- $message := join "," $messages -}}
{{- printf "%s" $message }}
{{- end -}}
