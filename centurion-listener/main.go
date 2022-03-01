package main

import (
	"flag"
	"log"

	sqs "ccs.sniff-n-fix.com/centurion-operator/pkg/sqs"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	centurionv1 "ccs.sniff-n-fix.com/centurion-operator/api/v1"
)

var centurionClient client.Client

func main() {
	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	queuename := flag.String("queuename", "", "full name of the SQS queue")

	flag.Parse()

	kubeConfigPath := *kubeconfig

	config, err := clusterConfig(kubeConfigPath)
	if err != nil {
		panic(err.Error())
	}

	centurionClient = GetClient(config)

	opts := zap.Options{
		Development: true,
	}
	sqsLog := zap.New(zap.UseFlagOptions(&opts))

	queueName := *queuename
	sqsMaxMessages := 1
	chnMessages := make(chan *awssqs.Message, sqsMaxMessages)
	go sqs.SqsListener(chnMessages, &queueName, sqsLog)

	for message := range chnMessages {
		sqs.HandleMessage(message, centurionClient)
		_ = sqs.DeleteMessage(message, &queueName)
	}
}

func clusterConfig(kubeConfigPath string) (*rest.Config, error) {
	if kubeConfigPath != "" {
		//  when not running in cluster
		return clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	}
	return rest.InClusterConfig()
}

func GetClient(kubeconfig *rest.Config) client.Client {
	scheme := runtime.NewScheme()
	centurionv1.AddToScheme(scheme)

	centurionClient, err := client.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatal(err)
		panic(err.Error())
	}
	return centurionClient
}
