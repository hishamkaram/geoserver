FROM tomcat:9.0.4-jre8
LABEL "MAINTAINER"="Hesham Karm<hishamwaleedkaram@gmail.com>"
ENV JAVA_HOME /opt/java
ARG GEOSERVER_VERSION=2.13.0
ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update \
    && apt-get install -y wget openssl ca-certificates \
    && cd /tmp \
    && wget --no-cookies \
    --no-check-certificate \
    -qO jdk8.tar.gz \
    --header "Cookie: oraclelicense=accept-securebackup-cookie" \
    https://javadl.oracle.com/webapps/download/GetFile/1.8.0_271-b09/61ae65e088624f5aaa0b1d2d801acb16/linux-i586/jdk-8u271-linux-x64.tar.gz \
    && tar xzf jdk8.tar.gz -C /opt \
    && mv /opt/jdk* /opt/java \
    && rm /tmp/jdk8.tar.gz \
    && update-alternatives --install /usr/bin/java java /opt/java/bin/java 100 \
    && update-alternatives --install /usr/bin/javac javac /opt/java/bin/javac 100

ENV JAVA_HOME /opt/java

RUN cd /tmp && \
    wget https://netix.dl.sourceforge.net/project/geoserver/GeoServer/${GEOSERVER_VERSION}/geoserver-${GEOSERVER_VERSION}-war.zip
RUN unzip /tmp/geoserver-${GEOSERVER_VERSION}-war.zip -d /tmp/geoserver
RUN mv /tmp/geoserver/geoserver.war /usr/local/tomcat/webapps
EXPOSE 8080