FROM openjdk:17-slim-bullseye

RUN apt-get update \
    && apt-get install -y net-tools \
    && rm -rf /var/lib/apt/lists/*

ARG MC_VERSION=1.18.2
WORKDIR /mc-server-data
ADD minecraft_server.${MC_VERSION}.jar /server.jar
ADD data/eula.txt eula.txt

CMD ["java", "-jar", "/server.jar", "nogui"]
