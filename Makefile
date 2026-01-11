# Variables
IMAGE_NAME = amos:v1
NAMESPACE = amos
DEPLOYMENT = amos-operator
SERVICE = amos-ui

.PHONY: run-server

run-server:
	@# Create cluster only if it doesn't exist
	kind get clusters | grep -q "^kind$$" || kind create cluster
	
	@# Build and load image
	docker build -t $(IMAGE_NAME) .
	kind load docker-image $(IMAGE_NAME)
	
	@# Apply configurations
	kubectl apply -f deploy/bootstrap.yaml
	
	@# Restart deployment to pick up new image and wait for ready status
	kubectl rollout restart deployment/$(DEPLOYMENT) -n $(NAMESPACE)
	kubectl rollout status deployment/$(DEPLOYMENT) -n $(NAMESPACE)
	
	@# Port-forward in the background so the terminal isn't locked
	@echo "Starting UI at http://localhost:8080"
	kubectl port-forward svc/$(SERVICE) 8080:80 -n $(NAMESPACE)
