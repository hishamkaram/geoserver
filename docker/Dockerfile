# GeoServer dev/test image.
# Base: Tomcat 10 + JDK 17 (Temurin). GeoServer 2.x runs on Servlet 4 (Tomcat 9/10).

ARG TOMCAT_TAG=10.1-jdk17-temurin
FROM tomcat:${TOMCAT_TAG}

LABEL maintainer="Hesham Karm<hishamwaleedkaram@gmail.com>"

ARG GEOSERVER_VERSION=2.28.0
ARG GEOSERVER_DOWNLOAD_URL=https://downloads.sourceforge.net/project/geoserver/GeoServer/${GEOSERVER_VERSION}/geoserver-${GEOSERVER_VERSION}-war.zip

ENV DEBIAN_FRONTEND=noninteractive \
    GEOSERVER_VERSION=${GEOSERVER_VERSION} \
    JAVA_OPTS="-Xms512m -Xmx2g -XX:+UseG1GC -Djava.awt.headless=true"

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        unzip \
    && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    rm -rf /usr/local/tomcat/webapps/*; \
    cd /tmp; \
    curl -fsSL --proto '=https' --tlsv1.2 -o geoserver.war.zip "${GEOSERVER_DOWNLOAD_URL}"; \
    unzip -q geoserver.war.zip -d /tmp/geoserver; \
    mv /tmp/geoserver/geoserver.war /usr/local/tomcat/webapps/geoserver.war; \
    rm -rf /tmp/geoserver /tmp/geoserver.war.zip

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=120s --retries=5 \
    CMD curl -fsS http://localhost:8080/geoserver/web/ || exit 1
