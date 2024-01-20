// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Code generated by ack-generate. DO NOT EDIT.

package main

import (
	"os"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcfg "github.com/aws-controllers-k8s/runtime/pkg/config"
	ackrt "github.com/aws-controllers-k8s/runtime/pkg/runtime"
	acktypes "github.com/aws-controllers-k8s/runtime/pkg/types"
	ackrtutil "github.com/aws-controllers-k8s/runtime/pkg/util"
	ackrtwebhook "github.com/aws-controllers-k8s/runtime/pkg/webhook"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlrt "sigs.k8s.io/controller-runtime"
	ctrlrtcache "sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlrtmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	ctrlrtwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	svctypes "github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
	svcresource "github.com/aws-controllers-k8s/eventbridge-controller/pkg/resource"
	svcsdk "github.com/aws/aws-sdk-go/service/eventbridge"

	_ "github.com/aws-controllers-k8s/eventbridge-controller/pkg/resource/archive"
	_ "github.com/aws-controllers-k8s/eventbridge-controller/pkg/resource/endpoint"
	_ "github.com/aws-controllers-k8s/eventbridge-controller/pkg/resource/event_bus"
	_ "github.com/aws-controllers-k8s/eventbridge-controller/pkg/resource/rule"

	"github.com/aws-controllers-k8s/eventbridge-controller/pkg/version"
)

var (
	awsServiceAPIGroup    = "eventbridge.services.k8s.aws"
	awsServiceAlias       = "eventbridge"
	awsServiceEndpointsID = svcsdk.EndpointsID
	scheme                = runtime.NewScheme()
	setupLog              = ctrlrt.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = svctypes.AddToScheme(scheme)
	_ = ackv1alpha1.AddToScheme(scheme)
}

func main() {
	var ackCfg ackcfg.Config
	ackCfg.BindFlags()
	flag.Parse()
	ackCfg.SetupLogger()

	managerFactories := svcresource.GetManagerFactories()
	resourceGVKs := make([]schema.GroupVersionKind, 0, len(managerFactories))
	for _, mf := range managerFactories {
		resourceGVKs = append(resourceGVKs, mf.ResourceDescriptor().GroupVersionKind())
	}

	if err := ackCfg.Validate(ackcfg.WithGVKs(resourceGVKs)); err != nil {
		setupLog.Error(
			err, "Unable to create controller manager",
			"aws.service", awsServiceAlias,
		)
		os.Exit(1)
	}

	host, port, err := ackrtutil.GetHostPort(ackCfg.WebhookServerAddr)
	if err != nil {
		setupLog.Error(
			err, "Unable to parse webhook server address.",
			"aws.service", awsServiceAlias,
		)
		os.Exit(1)
	}

	watchNamespaces := make(map[string]ctrlrtcache.Config, 0)
	namespaces, err := ackCfg.GetWatchNamespaces()
	if err != nil {
		for _, namespace := range namespaces {
			watchNamespaces[namespace] = ctrlrtcache.Config{}
		}
	}
	mgr, err := ctrlrt.NewManager(ctrlrt.GetConfigOrDie(), ctrlrt.Options{
		Scheme: scheme,
		Cache: ctrlrtcache.Options{
			Scheme:            scheme,
			DefaultNamespaces: watchNamespaces,
		},
		WebhookServer: &ctrlrtwebhook.DefaultServer{
			Options: ctrlrtwebhook.Options{
				Port: port,
				Host: host,
			},
		},
		Metrics:                 metricsserver.Options{BindAddress: ackCfg.MetricsAddr},
		LeaderElection:          ackCfg.EnableLeaderElection,
		LeaderElectionID:        "ack-" + awsServiceAPIGroup,
		LeaderElectionNamespace: ackCfg.LeaderElectionNamespace,
	})
	if err != nil {
		setupLog.Error(
			err, "unable to create controller manager",
			"aws.service", awsServiceAlias,
		)
		os.Exit(1)
	}

	stopChan := ctrlrt.SetupSignalHandler()

	setupLog.Info(
		"initializing service controller",
		"aws.service", awsServiceAlias,
	)
	sc := ackrt.NewServiceController(
		awsServiceAlias, awsServiceAPIGroup, awsServiceEndpointsID,
		acktypes.VersionInfo{
			version.GitCommit,
			version.GitVersion,
			version.BuildDate,
		},
	).WithLogger(
		ctrlrt.Log,
	).WithResourceManagerFactories(
		svcresource.GetManagerFactories(),
	).WithPrometheusRegistry(
		ctrlrtmetrics.Registry,
	)

	if ackCfg.EnableWebhookServer {
		webhooks := ackrtwebhook.GetWebhooks()
		for _, webhook := range webhooks {
			if err := webhook.Setup(mgr); err != nil {
				setupLog.Error(
					err, "unable to register webhook "+webhook.UID(),
					"aws.service", awsServiceAlias,
				)
			}
		}
	}

	if err = sc.BindControllerManager(mgr, ackCfg); err != nil {
		setupLog.Error(
			err, "unable bind to controller manager to service controller",
			"aws.service", awsServiceAlias,
		)
		os.Exit(1)
	}

	setupLog.Info(
		"starting manager",
		"aws.service", awsServiceAlias,
	)
	if err := mgr.Start(stopChan); err != nil {
		setupLog.Error(
			err, "unable to start controller manager",
			"aws.service", awsServiceAlias,
		)
		os.Exit(1)
	}
}
