FROM quay.io/deis/base:v0.3.6

RUN adduser --system \
	--shell /bin/bash \
	--disabled-password \
	--home /home/minio \
	--group \
	minio

COPY . /

RUN curl -f -SL https://dl.minio.io/client/mc/release/linux-amd64/archive/mc.RELEASE.2018-11-06T01-12-20Z -o /usr/bin/mc \
	&& chmod 755 /usr/bin/mc \
	&& mkdir /home/minio/.minio \
	&& chown minio:minio /home/minio/.minio
ADD https://dl.minio.io/server/minio/release/linux-amd64/minio.RELEASE.2018-11-06T01-01-02Z /bin/minio
RUN chmod 755 /bin/minio

USER minio

CMD ["/bin/boot"]
