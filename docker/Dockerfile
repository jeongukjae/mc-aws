FROM gcr.io/distroless/java17-debian11

ARG MC_VERSION=1.18.2

ADD minecraft_server.${MC_VERSION}.jar /server.jar

WORKDIR /mc-server-data
ADD data/eula.txt eula.txt

CMD ["/server.jar", "nogui"]
