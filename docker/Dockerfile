FROM tomcat:9.0.4-jre8
LABEL "MAINTAINER"="Cartologic Development Team"
ENV JAVA_HOME /usr/lib/jvm/default-java
ARG GEOSERVER_VERSION=2.13.0
RUN ln -s /usr/lib/jvm/java-8-openjdk-amd64 /usr/lib/jvm/default-java
RUN apt-get autoremove --purge -y openjdk-8-jre-headless && \
    echo oracle-java8-installer shared/accepted-oracle-license-v1-1 select true \
    | debconf-set-selections && \
    echo "deb http://ppa.launchpad.net/webupd8team/java/ubuntu xenial main" \
    > /etc/apt/sources.list.d/webupd8team-java.list && \
    apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys EEA14886 && \
    rm -rf /var/lib/apt/lists/* && \
    apt-get update && \
    apt-get install -y oracle-java8-installer oracle-java8-set-default && \
    ln -s --force /usr/lib/jvm/java-8-oracle /usr/lib/jvm/default-java

RUN cd /tmp && \
    wget https://netix.dl.sourceforge.net/project/geoserver/GeoServer/${GEOSERVER_VERSION}/geoserver-${GEOSERVER_VERSION}-war.zip
RUN unzip /tmp/geoserver-${GEOSERVER_VERSION}-war.zip -d /tmp/geoserver
RUN mv /tmp/geoserver/geoserver.war /usr/local/tomcat/webapps
EXPOSE 8080