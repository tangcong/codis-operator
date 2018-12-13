
# Image URL to use all building/pushing image targets
IMG ?= ccr.ccs.tencentyun.com/codis/codis-operator:latest

all: manager

# Run tests
test: generate fmt vet manifests
	echo ${PATH}
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager github.com/tangcong/codis-operator/cmd/manager

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./cmd/manager/main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kustomize build config/base > ./deploy/manager/deployment-pro.yml
	kustomize build config/dev > ./deploy/manager/deployment-dev.yml
	#cat ./config/crds/codis_v1alpha1_codiscluster.yaml >> ./deploy/manager/deployment.yml
	kubectl apply -f ./deploy/manager/deployment-dev.yml

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
	go generate ./pkg/... ./cmd/...

clean:
	rm -rf ./bin/
	rm -rf ./default.etcd
	rm -rf ./cover.out

# Build the docker image
docker-build:
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/base/manager_image_patch.yaml

# Push the docker image
docker-push:
	docker push ${IMG}
