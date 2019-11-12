FROM minio/mc:RELEASE.2019-09-24T01-36-20Z as mc

FROM minio/minio:RELEASE.2018-11-06T01-01-02Z

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
