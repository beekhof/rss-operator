E2E_TEST_SELECTOR=TestCreateCluster
TEST_NAMESPACE=testing
OPERATOR_IMAGE=quay.io/beekhof/galera-experiment:mac
KUBECONFIG=$$HOME/.kube/config

compile:
	PASSES=simple hack/test

clean: e2e-clean

e2e-clean:
#	kubectl -n testing delete svc,pods,sts --all
	-ssh root@192.168.124.10 -- kubectl -n testing delete svc,pods,sts,rss,crd --all
	sleep 10

e2e: e2e-clean
	@echo "Running tests: $(E2E_TEST_SELECTOR)"
	PASSES=e2e KUBECONFIG=$(KUBECONFIG) TEST_NAMESPACE=$(TEST_NAMESPACE) OPERATOR_IMAGE=$(OPERATOR_IMAGE) E2E_TEST_SELECTOR="$(E2E_TEST_SELECTOR)" hack/test 

build: 
	OPERATOR_IMAGE=$(OPERATOR_IMAGE) PASSES="prep simple build" hack/test

operator: 
	hack/build/operator/build
	cp _output/bin/rss-operator /usr/local/bin/rss-operator

all: build e2e-clean e2e

generated:
	-rm -rf pkg/generated
	-find pkg -name zz_generated.deepcopy.go #delete
	./hack/k8s/codegen/update-generated.sh 

t: target 

target:
	make -C apps/galera all

init: target deps generated all

deps:
	glide install --strip-vendor 