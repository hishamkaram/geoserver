FROM tomcat:9.0.4-jre9-slim
LABEL "MAINTAINER"="Hesham Karm<hishamwaleedkaram@gmail.com>"

ARG GEOSERVER_VERSION=2.18.1
ENV DEBIAN_FRONTEND noninteractive

RUN apt-get -qq update && \
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 648ACFD622F3D138 && \
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 0E98404D386FA1D9 && \
    apt-get -qq install -y wget && \
    rm -rf /var/lib/apt/lists/*

RUN cd /tmp && \
    wget --quiet https://downloads.sourceforge.net/project/geoserver/GeoServer/${GEOSERVER_VERSION}/geoserver-${GEOSERVER_VERSION}-war.zip

RUN unzip -qq /tmp/geoserver-${GEOSERVER_VERSION}-war.zip -d /tmp/geoserver && \
    mv /tmp/geoserver/geoserver.war /usr/local/tomcat/webapps && \
    rm -fr /tmp/geoserver

EXPOSE 8080