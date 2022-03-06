FROM gcr.io/distroless/java17-debian11

ARG MC_VERSION=1.18.2

WORKDIR /mc-server
ADD minecraft_server.${MC_VERSION}.jar server.jar
ADD eula.txt eula.txt
ADD server.properties server.properties

CMD ["server.jar", "nogui"]
