E2E_TEST_SELECTOR=TestCreateCluster
NS=test
TEST_APP=galera
KUBEHOST=192.168.124.10

IMAGE_REPO=quay.io
IMAGE_USER=beekhof

IMAGE=$(IMAGE_REPO)/$(IMAGE_USER)/rss-operator:latest
IMAGE_STATUS=https://$(IMAGE_REPO)/repository/$(IMAGE_USER)/rss-operator/status

PROJECT=github.com/beekhof/rss-operator
DOCKER_REPO_ROOT="/go/src/$(PROJECT)"

GIT_SHA=$(shell git rev-parse --short HEAD || echo GitNotFound)
export KUBECONFIG=$(HOME)/.kube/config
export GOPATH=$(HOME)/go
export GREP=grep --color=never

PKGS=$(shell go list ./cmd/... ./pkg/... | grep -v -e generated -e apis/clusterlabs/v1alpha1)
TEST_PKGS=$(shell go list ./test/... | grep -v -e generated -e apis/clusterlabs/v1alpha1)
GENERATE_CMD=docker run --rm -v "$(PWD):$(DOCKER_REPO_ROOT)" -w "$(DOCKER_REPO_ROOT)" gcr.io/coreos-k8s-scale-testing/codegen /go/src/k8s.io/code-generator/generate-groups.sh all $(PROJECT)/pkg/generated $(PROJECT)/pkg/apis clusterlabs:v1alpha1 --go-header-file "./hack/k8s/codegen/boilerplate.go.txt"

simple:
	$(GOPATH)/bin/gosimple $(PKGS)

build: 
	hack/build/operator/build

gitpush:
	git push

wait:
	date
	@echo "Waiting for the container to build..." 
	sleep 5
	-while [ "x$$(curl -s $(IMAGE_STATUS) | tr '<' '\n' | grep -v -e '>$$'  -e '^/' | sed 's/.*>//' | tail -n 1)" = xbuilding ]; do sleep 50; /bin/echo -n .; done
	curl -s $(IMAGE_STATUS) | tr '<' '\n' | grep -v -e ">$$"  -e '^/' | sed 's/.*>//' | tail -n 1
	date

push: dockerfile-checks gitpush wait

test-quick:
	gosimple $(TEST_PKGS)

publish: check build
	@echo "building container..."
	docker build --tag "${IMAGE}" -f hack/build/Dockerfile .
	@echo "Uploading to $(IMAGE)"
	docker push $(IMAGE)
	@echo "upload complete"

# Called from Dockerfile
install: dockerfile-checks build
	cp _output/bin/rss-operator /usr/local/bin/rss-operator

all: build push e2e-clean e2e

clean: e2e-clean

e2e-clean:
	# Delete stuff, wait for the pods to die, then delete the entire namespace
	-ssh root@$(KUBEHOST) -- kubectl -n $(NS) delete crd,deploy,rs,rss,sts,svc,pods --all
	while [ "x$$(kubectl -n $(NS) get po 2>/dev/null)" != "x" ]; do sleep 5; /bin/echo -n .; done
	-ssh root@$(KUBEHOST) -- kubectl delete ns $(NS)
	while [ "x$$(kubectl get ns $(NS) 2>/dev/null)" != "x" ]; do sleep 5; /bin/echo -n .; done


e2e: test-quick e2e-clean
	@echo "Running tests: $(E2E_TEST_SELECTOR)"
	PASSES=e2e TEST_NAMESPACE=$(NS) OPERATOR_IMAGE=$(IMAGE) E2E_TEST_SELECTOR="$(E2E_TEST_SELECTOR)" hack/test 

generated:
	-rm -rf pkg/generated
	-find pkg -name zz_generated.deepcopy.go #delete
	$(GENERATE_CMD)

dockerfile-checks: deps fmt unused simple

check: fmt unused simple verify-generated

verify-generated:
	$(GENERATE_CMD) --verify-only 

fmt:
	@echo "Checking gofmt..."
	for file in $(shell ./go-list.sh); do o=`gofmt -l -s -d $$file`; if [ "x$$o" != x ]; then echo "$$o"; exit 1; fi; done

unused:
	@echo "Checking unused..."
	$(GOPATH)/bin/unused $(PKGS)

init: target deps generated all

apps:
	make -C apps/galera all
	make -C apps/dummy all

deps:
	go get honnef.co/go/tools/cmd/gosimple
	go get honnef.co/go/tools/cmd/unused
	#glide install --strip-vendor 

ns:
	-kubectl create ns $(NS)
	-kubectl -n $(NS) create clusterrolebinding $(NS)-everything --clusterrole=cluster-admin --serviceaccount=$(NS):default

galera: 
	make TEST_APP=$@ NS=$@ clean test

dummy:
	make TEST_APP=$@ NS=$@ clean test

test: ns
	@echo "Loading apps/$(TEST_APP)/deployment.yaml into $(NS)"
	-kubectl -n $(NS) create -f apps/$(TEST_APP)/deployment.yaml
	@echo "Waiting for the operator to become active"
	while [ "x$$(kubectl -n $(NS) get po | grep rss-operator.*Running)" = x ]; do sleep 5; /bin/echo -n .; done
	kubectl -n $(NS) get po | $(GREP) rss-operator | awk '{print $$1}'
	kubectl -n $(NS) logs -f $$(kubectl -n $(NS) get po | $(GREP) rss-operator | awk '{print $$1}')

.PHONY: test
