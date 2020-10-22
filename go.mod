module github.com/3scale/marin3r

go 1.13

require (
	github.com/cncf/udpa/go v0.0.0-20200313221541-5f7e5dd04533
	github.com/envoyproxy/go-control-plane v0.9.5
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v0.1.0
	github.com/golang/protobuf v1.3.2
	github.com/jetstack/cert-manager v0.14.3
	github.com/operator-framework/operator-sdk v0.18.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.14.1
	google.golang.org/genproto v0.0.0-20191115194625-c23dd37a84c9
	google.golang.org/grpc v1.27.0
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kubernetes v1.13.0
	k8s.io/utils v0.0.0-20200324210504-a9aa75ae1b89
	sigs.k8s.io/controller-runtime v0.6.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.18.2 // Required by prometheus-operator
)
