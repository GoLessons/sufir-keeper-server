FROM minio/minio:latest AS minio

FROM minio/mc:latest AS mcsrc

FROM minio/minio:latest
COPY --from=mcsrc /usr/bin/mc /usr/bin/mc
COPY .docker/minio/entry.sh /usr/local/bin/minio-entry.sh
RUN chmod +x /usr/local/bin/minio-entry.sh

ENTRYPOINT ["/usr/local/bin/minio-entry.sh"]
CMD ["server", "/data", "--console-address", ":9001"]
