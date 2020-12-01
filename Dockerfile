FROM minio/mc:latest as mc


FROM drycc/go-dev:latest AS build
ARG LDFLAGS
ADD . /app
RUN export GO111MODULE=on \
  && cd /app \
  && CGO_ENABLED=0 go build -ldflags '-s' -o /usr/local/bin/boot boot.go


FROM minio/minio:RELEASE.2020-07-24T22-43-05Z

COPY rootfs /
COPY --from=mc /usr/bin/mc /bin/mc
COPY --from=build /usr/local/bin/boot /bin/boot

ENTRYPOINT ["/bin/boot"]
