build-docker-image:
	docker build -t lloydmeta/tasques:latest . -f docker/Dockerfile

push-docker-image:
	docker push lloydmeta/tasques
