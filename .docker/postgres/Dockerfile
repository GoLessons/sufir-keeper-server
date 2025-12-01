FROM postgres:16.3

# Install golang-migrate
RUN apt-get update && apt-get install -y curl \
    && curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz \
    && mv migrate /usr/local/bin/migrate \
    && chmod +x /usr/local/bin/migrate \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
