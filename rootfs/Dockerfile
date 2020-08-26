FROM minio/minio:RELEASE.2020-07-24T22-43-05Z

RUN wget https://dl.min.io/client/mc/release/linux-amd64/mc -O /bin/mc \
  && chmod +x /bin/mc

COPY . /

ENTRYPOINT ["/bin/boot"]
