# How to use docker

`docker pull postgres:12-alpine`

pull image

`docker run --name postgres12 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine`

create container

`docker exec -it postgres12 psql -U root`

`docker exec -it postgres12 /bin/sh`

access docker container via iteravctive tty and execute following command

`docker logs postgres12`

show docker logs

`docker ps`

show running containers

`docker ps -a`

show running and stopped containers

`docker stop postgres12`

stop containner

`docker start postgres12`

start container
