module ccs.sniff-n-fix.com/snf-operator

go 1.15

require (
	github.com/aws/aws-sdk-go v1.42.52 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.1.1
	github.com/aws/aws-sdk-go-v2/credentials v1.1.1
	github.com/aws/aws-sdk-go-v2/service/sqs v1.16.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.1.1
	github.com/go-logr/logr v1.2.0
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	k8s.io/apimachinery v0.23.4
	k8s.io/client-go v0.20.2
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
	sigs.k8s.io/controller-runtime v0.8.2
)
