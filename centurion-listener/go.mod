module ccs.sniff-n-fix.com/snf-operator/snf-listener

go 1.15

require (
	ccs.sniff-n-fix.com/snf-operator v0.0.0
	github.com/aws/aws-sdk-go-v2/service/sqs v1.1.1
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.2
)

replace ccs.sniff-n-fix.com/snf-operator => ../
