module github.com/3scale-ops/marin3r

go 1.16

require (
	github.com/MakeNowJust/heredoc v1.0.0
	github.com/davecgh/go-spew v1.1.1
	github.com/envoyproxy/go-control-plane v0.9.9
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/golang/protobuf v1.5.2
	github.com/goombaio/namegenerator v0.0.0-20181006234301-989e774b106e
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.17.0
	github.com/operator-framework/operator-lib v0.1.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/prometheus/common v0.31.1
	github.com/redhat-cop/operator-utils v1.3.2
	github.com/spf13/cobra v1.2.1
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c
	google.golang.org/grpc v1.38.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176
	sigs.k8s.io/controller-runtime v0.10.0
)

replace github.com/redhat-cop/operator-utils v1.3.2 => github.com/roivaz/operator-utils v0.0.0-20220121121047-9e3c33505230
