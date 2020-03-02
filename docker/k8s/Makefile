install-eck:
	kubectl apply -f https://download.elastic.co/downloads/eck/1.0.1/all-in-one.yaml

deploy:
	kubectl apply -f elasticsearch.yaml
	kubectl apply -f kibana.yaml
	kubectl apply -f apm.yaml
	kubectl apply -f tasques.yaml

teardown:
	kubectl delete \
		apm/tasques \
		kibana/tasques \
		elasticsearch/tasques \
		services/tasques \
		deployments/tasques

show-credentials:
	@echo "Tasques cluster elastic user password:"
	@kubectl get secret tasques-es-elastic-user -o=jsonpath='{.data.elastic}' | base64 --decode
	@echo "\n"
	@echo "APM secret:"
	@kubectl get secret/tasques-apm-token -o go-template='{{index .data "secret-token" | base64decode}}'
	@echo "\n"

