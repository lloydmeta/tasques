install-eck:
	kubectl apply -f https://download.elastic.co/downloads/eck/1.0.1/all-in-one.yaml

deploy-infra:
	kubectl apply -f elasticsearch.yaml
	kubectl apply -f kibana.yaml
	kubectl apply -f apm.yaml

deploy-tasques:
	@echo "Using config: ../../config/tasques.yaml"
	@echo "Make sure the file exists, and credentials are set properly (use show-credentials target) .."
	@rm -rf /tmp/tasques_config
	@cp -r ../../config /tmp/tasques_config
	kubectl apply -f tasques.yaml

teardown: teardown-infra teardown-tasques

teardown-infra:
	kubectl delete \
		apm/tasques \
		kibana/tasques \
		elasticsearch/tasques

teardown-tasques:
		kubectl delete services/tasques \
			deployments/tasques

show-credentials:
	@echo "Tasques cluster elastic user password:"
	@kubectl get secret tasques-es-elastic-user -o=jsonpath='{.data.elastic}' | base64 --decode
	@echo "\n"
	@echo "APM secret:"
	@kubectl get secret/tasques-apm-token -o go-template='{{index .data "secret-token" | base64decode}}'
	@echo "\n"

