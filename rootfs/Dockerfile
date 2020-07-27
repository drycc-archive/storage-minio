FROM minio/mc:RELEASE.2020-07-17T02-52-20Z as mc

FROM minio/minio:RELEASE.2020-07-24T22-43-05Z

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
