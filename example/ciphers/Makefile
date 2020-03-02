GO_WORKER_COUNT ?= 5

install-eck:
	@$(MAKE) install-eck -C k8s

k8s-deploy:
	@$(MAKE) deploy -C k8s

k8s-teardown:
	@$(MAKE) teardown -C k8s

k8s-show-credentials:
	@$(MAKE) show-credentials -C k8s

start-go-workers:
	@worker-go/start-workers ${GO_WORKER_COUNT}

start-cipher-server:
	@server/start-server