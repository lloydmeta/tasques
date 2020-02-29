DOCKER_TAG ?= latest

build-docker-image:
	docker build -t lloydmeta/tasques:${DOCKER_TAG} . -f docker/Dockerfile

push-docker-image:
	docker push lloydmeta/tasques:${DOCKER_TAG}
