FROM docker.io/drycc/go-dev:latest AS build
ARG LDFLAGS
ADD . /workspace
RUN export GO111MODULE=on \
  && cd /workspace \
  && CGO_ENABLED=0 init-stack go build -ldflags '-s' -o /usr/local/bin/boot boot.go

FROM docker.io/drycc/base:bullseye

COPY --from=build /usr/local/bin/boot /bin/boot

ENV DRYCC_UID=1001 \
  DRYCC_GID=1001 \
  DRYCC_HOME_DIR=/data \
  MC_VERSION="2022.04.01.23.44.48" \
  MINIO_VERSION="2022.04.01.03.41.39"


RUN groupadd drycc --gid ${DRYCC_GID} \
  && useradd drycc -u ${DRYCC_UID} -g ${DRYCC_GID} -s /bin/bash -m -d ${DRYCC_HOME_DIR} \
  && install-stack mc $MC_VERSION \
  && install-stack minio $MINIO_VERSION \
  && rm -rf \
      /usr/share/doc \
      /usr/share/man \
      /usr/share/info \
      /usr/share/locale \
      /var/lib/apt/lists/* \
      /var/log/* \
      /var/cache/debconf/* \
      /etc/systemd \
      /lib/lsb \
      /lib/udev \
      /usr/lib/`echo $(uname -m)`-linux-gnu/gconv/IBM* \
      /usr/lib/`echo $(uname -m)`-linux-gnu/gconv/EBC* \
  && mkdir -p /usr/share/man/man{1..8}

USER ${DRYCC_UID}
ENTRYPOINT ["init-stack", "/bin/boot"]
