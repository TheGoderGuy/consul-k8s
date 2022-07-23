package v1alpha1

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	capi "github.com/hashicorp/consul/api"
	admissionv1 "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:object:generate=false

type TerminatingGatewayServiceWebhook struct {
	client.Client
	ConsulClient *capi.Client
	Logger       logr.Logger
	decoder      *admission.Decoder
	//ConsulMeta   common.ConsulMeta
}

// NOTE: The path value in the below line is the path to the webhook.
// If it is updated, run code-gen, update subcommand/controller/command.go
// and the consul-helm value for the path to the webhook.
//
// NOTE: The below line cannot be combined with any other comment. If it is
// it will break the code generation.
//
// +kubebuilder:webhook:verbs=create;update,path=/mutate-v1alpha1-terminatinggatewayservices,mutating=true,failurePolicy=fail,groups=consul.hashicorp.com,resources=terminatinggatewayservices,versions=v1alpha1,name=mutate-terminatinggatewayservices.consul.hashicorp.com,sideEffects=None,admissionReviewVersions=v1beta1;v1

func (v *TerminatingGatewayServiceWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	var termGtwService TerminatingGatewayService
	var termGtwServiceList TerminatingGatewayServiceList
	err := v.decoder.Decode(req, &termGtwService)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Call validate first to ensure all the fields are validated before checking for service name duplicates.
	if err := termGtwService.Validate(); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if req.Operation == admissionv1.Create {
		v.Logger.Info("validate create", "name", termGtwService.KubernetesName())

		if err := v.Client.List(ctx, &termGtwServiceList); err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}

		for _, item := range termGtwServiceList.Items {
			// If any terminating gateway service resource has the same service name as this one, reject it.
			if item.Namespace == termGtwService.Namespace && item.ServiceInfo().ServiceName == termGtwService.ServiceInfo().ServiceName {
				return admission.Errored(http.StatusBadRequest,
					fmt.Errorf("an existing TerminatingGatewayService resource has the same service name `name: %s, namespace: %s`", item.ServiceInfo().ServiceName, termGtwService.Namespace))
			}
		}
	}

	return admission.Allowed(fmt.Sprintf("valid %s request", termGtwService.KubeKind()))
}

func (v *TerminatingGatewayServiceWebhook) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
