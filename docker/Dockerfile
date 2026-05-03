# GeoServer dev/test image.
#
# Base: Tomcat 9 + JDK 17 (Temurin). GeoServer 2.27 LTS / 2.28 stable use the
# javax.* servlet namespace (Servlet 4.x) and do not run on Tomcat 10/11
# (which moved to jakarta.*). GeoServer 3.0 will be the Tomcat 11 / Jakarta EE
# target and is tracked separately for v2 of this client library.

ARG TOMCAT_TAG=9-jdk17-temurin
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
