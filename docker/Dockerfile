# GeoServer dev/test image.
#
# Base: Tomcat 9 + JDK 17 (Temurin). GeoServer 2.27 LTS / 2.28 stable use the
# javax.* servlet namespace (Servlet 4.x) and do not run on Tomcat 10/11
# (which moved to jakarta.*). GeoServer 3.0 will be the Tomcat 11 / Jakarta EE
# target and is tracked separately for v2 of this client library.

ARG TOMCAT_TAG=9-jdk17-temurin
ARG TOMCAT_DIGEST=sha256:57536fa59652ece9df8d744749fa5a424136388f2eb934ddc8741918a51afd56
FROM tomcat:${TOMCAT_TAG}@${TOMCAT_DIGEST}

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

# Pre-extract the WAR so we can drop extension JARs into WEB-INF/lib.
# Tomcat will run the unpacked form just like the WAR.
RUN set -eux; \
    cd /usr/local/tomcat/webapps; \
    mkdir geoserver; \
    unzip -q geoserver.war -d geoserver; \
    rm geoserver.war

# Importer extension — required for the v2 SDK's `c.Imports`
# integration tests. Without it `GET /rest/imports` returns 404
# and the suite skips. Baked in here so CI exercises the full
# v2/rest/imports surface against a real server.
ARG IMPORTER_DOWNLOAD_URL=https://downloads.sourceforge.net/project/geoserver/GeoServer/${GEOSERVER_VERSION}/extensions/geoserver-${GEOSERVER_VERSION}-importer-plugin.zip
RUN set -eux; \
    cd /tmp; \
    curl -fsSL --proto '=https' --tlsv1.2 -o importer.zip "${IMPORTER_DOWNLOAD_URL}"; \
    mkdir importer-jars; \
    unzip -q -j importer.zip "*.jar" -d importer-jars; \
    cp importer-jars/*.jar /usr/local/tomcat/webapps/geoserver/WEB-INF/lib/; \
    rm -rf importer.zip importer-jars

# Monitor extension — required for the v2 SDK's `c.Monitor`
# integration tests. Without it `GET /rest/monitor/requests` returns
# 404. Baked in here so CI exercises the read-only monitoring REST
# surface against a real server. Default config (no persisted audit
# log) is fine for testing — the test only asserts the API contract.
ARG MONITOR_DOWNLOAD_URL=https://downloads.sourceforge.net/project/geoserver/GeoServer/${GEOSERVER_VERSION}/extensions/geoserver-${GEOSERVER_VERSION}-monitor-plugin.zip
RUN set -eux; \
    cd /tmp; \
    curl -fsSL --proto '=https' --tlsv1.2 -o monitor.zip "${MONITOR_DOWNLOAD_URL}"; \
    mkdir monitor-jars; \
    unzip -q -j monitor.zip "*.jar" -d monitor-jars; \
    cp monitor-jars/*.jar /usr/local/tomcat/webapps/geoserver/WEB-INF/lib/; \
    rm -rf monitor.zip monitor-jars

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=120s --retries=5 \
    CMD curl -fsS http://localhost:8080/geoserver/web/ || exit 1
