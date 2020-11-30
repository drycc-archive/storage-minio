FROM minio/mc:latest as mc


FROM minio/minio:RELEASE.2020-07-24T22-43-05Z

COPY . /
COPY --from=mc /usr/bin/mc /bin/mc

ENTRYPOINT ["/bin/boot"]
