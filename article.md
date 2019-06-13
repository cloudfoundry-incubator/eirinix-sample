# How to build Cloud Foundry Eirini Extensions with Eirinix

Working with Eirini and Cloud Foundry is really exciting.

The possibility to leverage directly the features of Kubernetes with the Cloud Foundry
Ecosystem opens up a new wide scenarios and completely new ecosystems around Eirini.

When you want to simply provide new features around Eirini, probably means you need to interact with Kubernetes API.
This can be a complex and repetitive work, and which usually takes time from the Design to the testing process (expecially in real-world scenarios).

At SUSE we've built a framework around Eirini and Kubernetes, that we called "**Eirinix**" to provide a common way to build Extensions around Eirini, using Kubernetes API.

At beginning, the components that are now part of Eirinix, were used to provide persistence support to Cloud Foundry Eirini apps with the usage of Persistent Volume Claims.

Eirinix allows you to focus on the logic of your extension instead of loosing time with writing common bootstrap operation to get your logic to work in a real kubernetes cluster.

In this article we will see how to build a simple Extension for Eirini with Eirinix and how to test it locally.

---

**Note**: The full sample that will be realized in this article can be found in the _eirinix-sample_ Github repository [1].

## Mutating webhooks

When dealing with the problem on how-to extend Eirini in a Kubernetes "native" way, we re-used part of the solutions already implemented by what then became the **Quarks Project in Cloud Foundry** [2].

Eirinix is building a Kubernetes Mutating webhook [3][4] from each Extension, so each one can be also seen as a standalone Kubernetes operator targetting Eirini applications.

For semplification, we can look at mutating webhooks as a way to intercept Kubernetes requests dynamically, and Eirinix as a way to handle this setup step for you.

When an apiserver receives a request that matches one of the rules provided by the registered mutating webhooks in a kubernetes cluster, the apiserver sends an admissionReview request to webhook, and the webhook replies with a response patch.

The core of an Eirini Extension is to provide the operations as a set of patches against a POD which is about to run in the Kubernetes cluster. The 
Eirinix framework will convert the Extension to a webhook server which serves patches for the Kubernetes API server.

## Create your first Extension

We are going to create our first Extension and we will later run it on Minikube. Let's say that we want to create an Extension that will inject a new environment variable to every application that is being pushed on a Cloud Foundry instance.

If the user pushes a general application X, we want that at the runtime the application has a new environment variable, let's call it ```EXAMPLE```, in their pod definition.

The same approach could be taken, for example to inject sidecar containers for each application.

What the extension should do then, is just modifying the POD definition, going over all the existing containers, and append the environment variable.


### Requirements

* Golang (or Docker)
* Minikube (for testing)

Let's create a directory in the system for our extension, here in after called _eirinix-helloworld_, and we assume the repository path ideally is _github.com/eirinix/eirinix-helloworld_,
it basically will be composed of two files: one which is our ```main.go```, and will run the extension, and one that holds the Extension logic ```hello/helloworld.go```.

That's our tree structure:

```
eirinix-heloworld
├── hello
│   └── helloworld.go
└── main.go

1 directory, 2 files
```

### 1) Create hello/helloworld.go

We will sketch up the logic of our Extension first, so let's create ```hello/helloworld.go``` and populate it with the following content:

```golang
package hello

import (
    "context"
    "errors"
    "net/http"

    eirinix "github.com/SUSE/eirinix"
    corev1 "k8s.io/api/core/v1"
    "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
    "sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

type Extension struct{}

func (ext *Extension) Handle(ctx context.Context, eiriniManager eirinix.Manager, pod *corev1.Pod, req types.Request) types.Response {

    if pod == nil {
        return admission.ErrorResponse(http.StatusBadRequest, errors.New("No pod could be decoded from the request"))
    }
    podCopy := pod.DeepCopy()

    for i := range podCopy.Spec.Containers {
        c := &podCopy.Spec.Containers[i]
        c.Env = append(c.Env, corev1.EnvVar{Name: "EXAMPLE", Value: "Eirinix is awesome!"})
    }
    return admission.PatchResponse(pod, podCopy)
}
```

An Eirinix Extension is a structure which has a defined ```Handle``` method. It needs to accept ```ctx context.Context, eiriniManager eirinix.Manager, pod *corev1.Pod, req types.Request``` , which are the Kubernetes API requests plus the Eirinix Manager interface, and returns a ```types.Response``` structure, which is a response that Kubernetes can understand. That's the only requested method by the **Eirinix Extension interface** [5].

In our case, our response will be derived from a difference between two pods: the original POD  that we receive from the request, and how we want the POD to actually look like after our changes.

In ```podCopy := pod.DeepCopy()``` we are taking a copy of the original POD definition from the request, and we create a new pod data structure (this is not happening in runtime, we aren't duplicating any real POD running in the cluster!).

Every change that we want to commit over a POD will be done to ```podCopy```, and later on we will compute the patchset from the two POD structures by calling ``` admission.PatchResponse(pod, podCopy)```, where ```podCopy``` is including our changes.

To keep things simple, we will create a ```main.go``` files which just starts our Extension with hardcoded cluster connection values.

### 2) Create main.go

Our ```main.go``` will be very simple, we just need at this point to register our Extension, and start the Eirinix Extension Manager.

```golang
package main

import (

    "os"

    eirinix "github.com/SUSE/eirinix"
    helloworld "github.com/eirinix/eirinix-helloworld/hello"
)

func main() {
    x := eirinix.NewManager(
            eirinix.ManagerOptions{
                Namespace:           "default",
                Host:                "10.0.2.2",
                Port:                3000,
                KubeConfig:          os.Getenv("KUBECONFIG"),
                OperatorFingerprint: "eirini-x-helloworld",
            })

    x.AddExtension(&helloworld.Extension{})
    x.Start()
}
````

Here we create a new Eirinix Manager passing *eirinix.ManagerOptions* [6] with the connection parameters. ```Namespace``` is set to default, it means that we expect Eirini Application pods to be pushed here. ```Host``` is the ip, or hostname where the Manager server is listening to. You need to set this value to a IP/domain which is reachable from the Kubernetes API server.

In our case, we will set it to ```10.0.2.2``` as we will use later Minikube to test our extension (in Minikube, this is the IP of the host reachable from the Kubeapi server). 

```Port``` it is the port where the Manager server is listening to. Can be set to any value, just make sure that port is reachable if you run this inside a cluster or not declined by any fw rule.

```KubeConfig``` accepts a path to a Kubernetes config file, or otherwise omit it for in-cluster connection. We will read it from the environment variable ```KUBECONFIG``` in this case.

```OperatorFingerprint``` It's a unique identifier for your operator. It can be any arbitrary value. You need to set this only in the case you are planning to run more than one manager in the same cluster (in different PODs).

With ```x.AddExtension(&helloworld.Extension{})``` we are adding our Extension to the manager, later on the Manager will build a mutating webhook from it and will run it to serve requests. You can add more extension in the same process.

```x.Start()``` starts the main loop. It returns an error in case there were runtime issues (connection failures to k8s, etc. )

### 3) Build it!

    $> echo 'module github.com/eirinix/eirinix-helloworld' > go.mod
    $> go get github.com/SUSE/eirinix
    $> go build

If you don't have Golang installed in your machine, you can build your project by running the steps above in a Golang container e.g.:

    $> docker run -v $PWD:/eirinix-helloworld --workdir /eirinix-helloworld --rm -ti golang /bin/bash

We should have now a new binary in our project folder, ```eirinix-helloworld```. 

### 4) Run

It's time to start minikube

    $> minikube start

Now, we can finally run our extension:

    $> ./eirinix-helloworld
    2019-06-12T09:41:06.435+0200    INFO    eirinix-helloworld/main.go:26       Starting 0.0.1 with namespace default
    2019-06-12T09:41:06.437+0200    INFO    config/getter.go:91 Using kube config '/home/user/.kube/config'
    2019-06-12T09:41:06.437+0200    INFO    config/checker.go:36 Checking kube config
    2019-06-12T09:41:07.401+0200    INFO    ctxlog/context.go:51 Creating webhook server certificate
   


### 5) (Mock) Test it

Let's try to create a POD that looks like an Eirini app. We will spawn a busybox image that will run enough to let us inspect it. In another terminal run:

    $> cat <<EOF | kubectl apply -f -
    apiVersion: v1
    kind: Pod
    metadata:
        name: eirini-fake-app
        labels:
           source_type: APP
    spec:
        containers:
            - image: busybox:1.28.4
              command:
                - sleep
                - "3600"
              name: eirini-fake-app
        restartPolicy: Always
    EOF

And after few seconds, inspect the pod, we should be able to see our environment variable setted by the Extension:

    $> kubectl describe pod eirini-fake-app

    ...
    Status:             Running
    IP:                 172.17.0.4
    Containers:             
    eirini-fake-app:      
        Container ID:  docker://244dc2a2c7a36d078549b0d43899fff60bdb3cd53bf13927d3a6024b42a0ac01
        Image:         busybox:1.28.4
        Image ID:      docker-pullable://busybox@sha256:141c253bc4c3fd0a201d32dc1f493bcf3fff003b6df416dea4f41046e0f37d47
        Port:          <none>           
        Host Port:     <none>
        Command:               
        sleep            
        3600                                                      
        State:          Running                                       
        Started:      Thu, 13 Jun 2019 08:49:10 +0200
        Ready:          True                             
        Restart Count:  0                                
        Environment:                                                                                       
        EXAMPLE:  Eirinix is awesome!                                                                      
        Mounts:                                                    
        /var/run/secrets/kubernetes.io/serviceaccount from default-token-m97vc (ro)
    ...

You can verify that the extension is working correctly by checking that the environment variable ```EXAMPLE``` is present in the Environment section in the container's POD.

## Run your extension in a Cloud Foundry deployment

Eirinix allows you to run your component inside a kubernetes cluster without caring about the connection details - in the example above, we would need to omit ```KubeConfig``` in the Eirinix Manager Options to achieve that.

In a real-use case, at SUSE, when developing the Eirini persistence support, we consumed the extension in the eirini-bosh-release [7][8].

You can also test your Extension inside a kubernetes cluster without a full bosh/Cloud Foundry deployment, see [9] for a step-by-step example.


Few notes to keep in mind while doing so:

- Depending on the Kubernetes cluster setup for internal resolution, the Kubernetes API server could be not able to resolve internal DNS names (see e.g. [10] ). Prefer IPs over domain name in the Host section of the Manager option. This brings the disadvantage that migrations of the extension to different ips involves the recreation of the webhook certificates, and deletion of old ones.
Keep in mind that the ```Host``` parameter is not only where the service is binding to, but it is also the address advertized to the Kubernetes API server for contacting the webhook server ( do not bind to ```0.0.0.0``` )
- Allocating a new service with the Kubernetes API and consuming the reserved ```ClusterIP``` is the most straightforward way to allocate a fixed IP for the service, to avoid certificates regeneration.
- Among the Eirinix Manager options you can set up a default Failure policy (see [6][11]). If set to fail, whatever error could occur in your Extension would causes the admission to fail, and no POD would be started - by default, if not specified, Eirinix uses Fail as default policy.

### A "pluggable" Eirini Ecosystem

With Extensions defined as separated logical pieces we gain the additional value that we can just plug the features that we need to our deployment. 
Also, with an _extensible_ approach, features can be easily re-used and shared within the community without having impact on the development of Eirini core-features.

## References

- [1] *Eirinix Sample Extension* - https://github.com/SUSE/eirinix-sample
- [2] *Quarks Project in Cloud Foundry* - https://www.cloudfoundry.org/project-quarks/
- [3] *"Using Admission Controllers"* - https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook
- [4] *"Dynamic Admission Control"* - https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#what-are-admission-webhooks
- [5] *Eirinix Extension interface* - https://godoc.org/github.com/SUSE/eirinix#Extension
- [6] *Eirinix ManagerOptions* - https://godoc.org/github.com/SUSE/eirinix#ManagerOptions
- [7] *Eirini bosh release, eirini-extensions job* - https://github.com/cloudfoundry-community/eirini-bosh-release/tree/master/jobs/eirini-extensions
- [8] *Eirini bosh release, eirini-extensions package spec*- https://github.com/cloudfoundry-community/eirini-bosh-release/tree/master/packages/eirini-extensions
- [9] *Minikube In-cluster Example*- https://github.com/SUSE/eirinix-sample#minikube-in-cluster-example
- [10] *"apiserver pod is not able to resolve internal DNS: Name or service not known"* - https://github.com/kubernetes/minikube/issues/3772
- [11] *Admission Control Failure Policy* - https://godoc.org/k8s.io/api/admissionregistration/v1beta1#FailurePolicyType