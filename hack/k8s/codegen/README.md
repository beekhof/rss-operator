## How to use codegen

Dependencies:
- Docker

In repo root dir, run:

```sh
./hack/k8s/codegen/update-generated.sh
```

It should print:

```
Generating deepcopy funcs
Generating clientset for galera:v1alpha1 at github.com/beekhof/galera-operator/pkg/generated/clientset
Generating listers for galera:v1alpha1 at github.com/beekhof/galera-operator/pkg/generated/listers
Generating informers for galera:v1alpha1 at github.com/beekhof/galera-operator/pkg/generated/informers
```
