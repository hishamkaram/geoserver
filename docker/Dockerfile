FROM tomcat:jdk8-adoptopenjdk-hotspot
LABEL "MAINTAINER"="Hesham Karm<hishamwaleedkaram@gmail.com>"
ARG GEOSERVER_VERSION=2.13.0
ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update \
    && apt-get install -y unzip wget openssl ca-certificates

RUN cd /tmp && wget --no-check-certificate https://downloads.sourceforge.net/project/geoserver/GeoServer/${GEOSERVER_VERSION}/geoserver-${GEOSERVER_VERSION}-war.zip
RUN unzip /tmp/geoserver-${GEOSERVER_VERSION}-war.zip -d /tmp/geoserver
RUN mv /tmp/geoserver/geoserver.war /usr/local/tomcat/webapps
EXPOSE 8080