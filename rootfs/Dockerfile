FROM minio/mc:RELEASE.2020-07-17T02-52-20Z as mc

FROM minio/minio:RELEASE.2020-07-24T22-43-05Z

COPY --from=mc /usr/bin/mc /usr/bin/mc

COPY . /

ENTRYPOINT ["/bin/boot"]
