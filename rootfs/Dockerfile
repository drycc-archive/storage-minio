FROM minio/mc:RELEASE.2019-05-23T01-33-27Z as mc

FROM minio/minio:RELEASE.2019-05-23T00-29-34Z

COPY --from=mc /usr/bin/mc /usr/bin/mc

RUN adduser \
       -s /bin/sh \
       -D  \
       -h /home/minio \
       minio \
       minio

COPY . /

RUN mkdir /home/minio/.minio \
        && chown minio:minio /home/minio/.minio

USER minio

ENTRYPOINT ["/bin/boot"]
