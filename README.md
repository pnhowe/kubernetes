# t3kton Kubernetes Operator

A Kubernetes operator that exposes existing [Contractor](https://github.com/t3kton/contractor) Structures as
`Structure` custom resources. It does not create Structures or Foundations in Contractor, it manages
the ones that already exist: building them, destroying them, changing their blueprint and config values, and reporting
their state and job progress back into Kubernetes.

## The Structure resource

```yaml
apiVersion: contractor.t3kton.com/v1
kind: Structure
metadata:
  name: structure-sample
spec:
  id: 3                         # ID of the Structure in Contractor, required, immutable
  state: built                  # planned | built, defaults to the current state in Contractor
  blueprint: test-structure-base # defaults to the current blueprint in Contractor
  configValues:                 # defaults to the current values in Contractor
    myval: stuff
    thisint: 123
  consumerRef: {}               # optional, free form reference to whatever is using this Structure, not used by the operator
```

The defaults are filled in by the mutating webhook at creation, so `spec: {id: 3}` will adopt the Structure as it is
in Contractor right now.

`status` shows the current `state`, `blueprint`, `hostname`, `configValues`, the `foundation` and its blueprint,
and the running `job` (script, state, message, progress, time remaining), `kubectl get structures` shows the highlights.
The operator emits events for job creation and completion, and for spec syncs in follow mode.

### What the operator does (sync mode `enforce`, the default)

The Structure resource is the source of truth, and Contractor is changed to match it:

- `configValues` are written straight to Contractor.
- `state: built` starts a `create` job, `state: planned` starts a `destroy` job.
- Changing the `blueprint` updates it in Contractor, which only allows that while the Structure is planned. The validation
  webhook enforces the same rule: to change a built Structure's blueprint, set `state: planned`, wait for the destroy job
  to finish, change the `blueprint`, then set `state: built` again.
- While a job is running the operator only watches, and the job's progress is shown in `status.job`.

This includes changes made directly in Contractor (for example from the UI), they are reverted to match the
spec. The operator checks idle Structures every `--resync-interval` (default 5 minutes), so changes are noticed
even if nothing happens in Kubernetes.

### Following Contractor (sync mode `follow`)

With `--sync-mode=follow` (or `SYNC_MODE=follow`) Contractor is the source of truth. The operator never changes
anything in Contractor, never starts jobs, and instead updates the spec of the Structure resources to match
Contractor (`state`, then `blueprint`, `configValues`) once there is no job running. Edits made to the spec are overwritten
at the next resync. This is for when the Structures are managed elsewhere and Kubernetes only needs to know about them.

The mode is global, to stop an individual Structure from being managed, delete its `Structure` resource,
deleting the resource never destroys or changes anything in Contractor. (The webhook does refuse to delete
a Structure that is, or is targeted to be, `built`, or has a job, change the state to `planned` first.)

## Configuration

| Flag | Environment | Default | |
|---|---|---|---|
| `--contractor-host` | `CONTRACTOR_HOST` | `http://contractor` | Contractor's URL |
| `--contractor-username` | `CONTRACTOR_USERNAME` | `k8s` | |
| `--contractor-password` | `CONTRACTOR_PASSWORD` | `k8s` | |
| `--contractor-proxy` | `CONTRACTOR_PROXY` | | proxy to reach Contractor through |
| `--sync-mode` | `SYNC_MODE` | `enforce` | `enforce` or `follow` |
| `--resync-interval` | | `5m` | `0` disables the periodic resync |

The flags override the environment. The deployment in `config/manager/manager.yaml` reads the connection info from
a Secret named `contractor-credentials` (it is not part of the kustomization, so create it by hand in the
operator's namespace):

```sh
kubectl create namespace t3kton-contractor
kubectl -n t3kton-contractor create secret generic contractor-credentials \
  --from-literal=host=http://contractor.example.com \
  --from-literal=username=k8s \
  --from-literal=password=CHANGEME
```

A template is in `config/samples/contractor-credentials-secret.yaml` (it is deliberately not in the samples
kustomization, so the placeholders are never applied by accident).

The Contractor user needs to be able to read Structures, Foundations, BluePrints and Jobs, and in `enforce` mode update
Structures and run their create and destroy jobs. The operator logs in at startup and refreshes its token as needed, if
Contractor is unreachable it retries with backoff.

The webhooks (defaulting and validation) need a TLS certificate, the deployment uses cert-manager for that.

## Getting Started

### Prerequisites
- go version v1.26+
- docker version 17.03+.
- kubectl and access to a Kubernetes v1.32+ cluster, with [cert-manager](https://cert-manager.io) installed.

### Deploy to a cluster

```sh
make docker-build docker-push IMG=<some-registry>/t3kton-operator:tag
make install                                   # the CRDs
make deploy IMG=<some-registry>/t3kton-operator:tag
kubectl -n t3kton-contractor create secret ... # see Configuration above
kubectl apply -k config/samples/               # edit the ID to match a real Structure first
```

`make deploy` creates the `t3kton-contractor` namespace. The operator pod will not start until the Secret exists
(it shows `CreateContainerConfigError`), it starts on its own once the Secret is created, no restart is needed.

If you encounter RBAC errors, you may need to be logged in as cluster-admin.

### Uninstall

```sh
kubectl delete -k config/samples/   # the Structure resources, Contractor is not changed
make undeploy                       # the operator
make uninstall                      # the CRDs
```

### Distribution

`make build-installer IMG=<some-registry>/t3kton-operator:tag` makes `dist/install.yaml`, with everything built by kustomize,
which can be applied with `kubectl apply -f`.

## Local development and testing

```sh
mkdir -p /tmp/k8s-webhook-server/serving-certs
openssl req -x509 -newkey rsa:4096 -keyout /tmp/k8s-webhook-server/serving-certs/tls.key -out /tmp/k8s-webhook-server/serving-certs/tls.crt -sha256 -days 3650 -nodes -subj "/C=NA/ST=NA/L=NA/O=NA/OU=NA/CN=NA"

kind create cluster
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.0/cert-manager.yaml  # or any recent release
# from the root of this repository
make install

# either run the operator in the cluster (needs the contractor-credentials Secret)
make undeploy ; export VER=0.0.12 ; make docker-build IMG=t3kton:$VER && kind load docker-image t3kton:$VER --name kind && make deploy IMG=t3kton:$VER
# or run it from your machine (webhook certs above are used), pointing it at your Contractor
CONTRACTOR_HOST=http://172.19.0.1:8888 CONTRACTOR_USERNAME=root CONTRACTOR_PASSWORD=root make run

kubectl create -f config/samples/contractor_v1_structure.yaml   # edit the id first
```

The unit and controller tests are run with `make test`, the e2e tests (which need kind) with `make test-e2e`, and the linter with `make lint`.

## Contributing

The project was originally scaffolded with [Kubebuilder](https://book.kubebuilder.io/introduction.html), but is now maintained by hand.
The CRD, RBAC and webhook manifests are generated from the `+kubebuilder:` markers in the Go code by `make manifests generate`, run it
after changing the API types or markers. Run `make help` for all the `make` targets.

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
