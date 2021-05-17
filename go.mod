module github.com/3scale-ops/marin3r

go 1.15

require (
	github.com/MakeNowJust/heredoc v1.0.0
	github.com/davecgh/go-spew v1.1.1
	github.com/envoyproxy/go-control-plane v0.9.8
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.3.0
	github.com/golang/protobuf v1.4.3
	github.com/goombaio/namegenerator v0.0.0-20181006234301-989e774b106e
	github.com/onsi/ginkgo v1.15.2
	github.com/onsi/gomega v1.11.0
	github.com/operator-framework/operator-lib v0.1.0
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/prometheus/common v0.10.0
	github.com/redhat-cop/operator-utils v1.1.2
	github.com/spf13/cobra v1.1.3
	google.golang.org/genproto v0.0.0-20200701001935-0939c5918c31
	google.golang.org/grpc v1.30.0
	google.golang.org/protobuf v1.25.0
	k8s.io/api v0.20.0
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v0.20.0
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	sigs.k8s.io/controller-runtime v0.7.2
)

replace github.com/redhat-cop/operator-utils v1.1.2 => github.com/roivaz/operator-utils v1.1.3-0.20210518155433-82fe9bc469ab
