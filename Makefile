DOCKER_TAGS ?= latest

build-docker-image:
	@tag_string=''; \
	 for tag in ${DOCKER_TAGS}; do tag_string="$$tag_string -t lloydmeta/tasques:$$tag"; done; \
	 docker build $$tag_string . -f docker/Dockerfile

push-docker-image:
	@for tag in ${DOCKER_TAGS}; do docker push lloydmeta/tasques:$$tag; done;
