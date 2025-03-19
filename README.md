# cevichedbsync-operator
A Kubernetes operator to automate PostgreSQL database dumps and sync with Git repositories.


## Description
The Ceviche Database Sync Operator watches PostgreSQL StatefulSets in Kubernetes and automatically creates, manages, and synchronizes database dumps to Git repositories. This enables easy backup, versioning, and recovery of database content.


### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/cevichedbsync-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/cevichedbsync-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

### Usage
1. Create a secret with Git credentials
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: git-credentials
  namespace: postgres
type: Opaque
data:
  username: <base64-encoded-git-username>
  password: <base64-encoded-git-token>
```

2. Create a Secret with database credentials:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-credentials
  namespace: postgres
type: Opaque
data:
  username: <base64-encoded-db-username>
  password: <base64-encoded-db-password>
  database: <base64-encoded-db-name>
  port: <base64-encoded-port>
```

3. Create a PostgresSync resource:
```yaml
apiVersion: ceviche.jcroyoaun.io/v1alpha1
kind: PostgresSync
metadata:
  name: sample-postgres-sync
  namespace: postgres
spec:
  repositoryURL: "https://github.com/yourusername/your-repo.git"
  databaseDumpPath: "dumps"
  gitCredentials:
    secretName: git-credentials
  databaseCredentials:
    secretName: postgres-credentials
  databaseService:
    name: postgres-svc
    namespace: postgres
  statefulSetRef:
    name: postgres
  dumpOnWebhook: false
```

4. Hit the Webhook endpoint to trigger a dump
```yaml
curl -X POST http://<operator-service>:8082/dump/postgres/sample-postgres-sync

```

NOTE: In need of port-forwarding pod directly to hit endpoint

## Contributing
Send me a DM on x.com/@jcroyoaun

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

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

