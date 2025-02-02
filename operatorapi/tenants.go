// This file is part of MinIO Console Server
// Copyright (c) 2021 MinIO, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package operatorapi

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	utils2 "github.com/minio/console/pkg/utils"

	"github.com/dustin/go-humanize"

	"github.com/minio/console/restapi"

	"github.com/minio/console/operatorapi/operations/operator_api"

	"github.com/minio/console/pkg/auth/utils"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"

	"github.com/minio/console/cluster"
	"github.com/minio/madmin-go"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/swag"
	"github.com/minio/console/models"
	"github.com/minio/console/operatorapi/operations"
	miniov2 "github.com/minio/operator/pkg/apis/minio.min.io/v2"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sJson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type imageRegistry struct {
	Auths map[string]imageRegistryCredentials `json:"auths"`
}

type imageRegistryCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth"`
}

func registerTenantHandlers(api *operations.OperatorAPI) {
	// Add Tenant
	api.OperatorAPICreateTenantHandler = operator_api.CreateTenantHandlerFunc(func(params operator_api.CreateTenantParams, session *models.Principal) middleware.Responder {
		resp, err := getTenantCreatedResponse(session, params)
		if err != nil {
			return operator_api.NewCreateTenantDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewCreateTenantOK().WithPayload(resp)
	})
	// List All Tenants of all namespaces
	api.OperatorAPIListAllTenantsHandler = operator_api.ListAllTenantsHandlerFunc(func(params operator_api.ListAllTenantsParams, session *models.Principal) middleware.Responder {
		resp, err := getListAllTenantsResponse(session, params)
		if err != nil {
			return operator_api.NewListTenantsDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewListTenantsOK().WithPayload(resp)

	})
	// List Tenants by namespace
	api.OperatorAPIListTenantsHandler = operator_api.ListTenantsHandlerFunc(func(params operator_api.ListTenantsParams, session *models.Principal) middleware.Responder {
		resp, err := getListTenantsResponse(session, params)
		if err != nil {
			return operator_api.NewListTenantsDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewListTenantsOK().WithPayload(resp)

	})
	// Detail Tenant
	api.OperatorAPITenantDetailsHandler = operator_api.TenantDetailsHandlerFunc(func(params operator_api.TenantDetailsParams, session *models.Principal) middleware.Responder {
		resp, err := getTenantDetailsResponse(session, params)
		if err != nil {
			return operator_api.NewTenantDetailsDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewTenantDetailsOK().WithPayload(resp)

	})

	// Tenant Security details
	api.OperatorAPITenantSecurityHandler = operator_api.TenantSecurityHandlerFunc(func(params operator_api.TenantSecurityParams, session *models.Principal) middleware.Responder {
		resp, err := getTenantSecurityResponse(session, params)
		if err != nil {
			return operator_api.NewTenantSecurityDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewTenantSecurityOK().WithPayload(resp)

	})

	// Update Tenant Security configuration
	api.OperatorAPIUpdateTenantSecurityHandler = operator_api.UpdateTenantSecurityHandlerFunc(func(params operator_api.UpdateTenantSecurityParams, session *models.Principal) middleware.Responder {
		err := getUpdateTenantSecurityResponse(session, params)
		if err != nil {
			return operator_api.NewUpdateTenantSecurityDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewUpdateTenantSecurityNoContent()

	})

	// Delete Tenant
	api.OperatorAPIDeleteTenantHandler = operator_api.DeleteTenantHandlerFunc(func(params operator_api.DeleteTenantParams, session *models.Principal) middleware.Responder {
		err := getDeleteTenantResponse(session, params)
		if err != nil {
			return operator_api.NewDeleteTenantDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewDeleteTenantNoContent()

	})

	// Delete Pod
	api.OperatorAPIDeletePodHandler = operator_api.DeletePodHandlerFunc(func(params operator_api.DeletePodParams, session *models.Principal) middleware.Responder {
		err := getDeletePodResponse(session, params)
		if err != nil {
			return operator_api.NewDeletePodDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewDeletePodNoContent()

	})

	// Update Tenant
	api.OperatorAPIUpdateTenantHandler = operator_api.UpdateTenantHandlerFunc(func(params operator_api.UpdateTenantParams, session *models.Principal) middleware.Responder {
		err := getUpdateTenantResponse(session, params)
		if err != nil {
			return operator_api.NewUpdateTenantDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewUpdateTenantCreated()
	})

	// Add Tenant Pools
	api.OperatorAPITenantAddPoolHandler = operator_api.TenantAddPoolHandlerFunc(func(params operator_api.TenantAddPoolParams, session *models.Principal) middleware.Responder {
		err := getTenantAddPoolResponse(session, params)
		if err != nil {
			return operator_api.NewTenantAddPoolDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewTenantAddPoolCreated()
	})

	// Get Tenant Usage
	api.OperatorAPIGetTenantUsageHandler = operator_api.GetTenantUsageHandlerFunc(func(params operator_api.GetTenantUsageParams, session *models.Principal) middleware.Responder {
		payload, err := getTenantUsageResponse(session, params)
		if err != nil {
			return operator_api.NewGetTenantUsageDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewGetTenantUsageOK().WithPayload(payload)
	})

	// Get Tenant Logs
	api.OperatorAPIGetTenantLogsHandler = operator_api.GetTenantLogsHandlerFunc(func(params operator_api.GetTenantLogsParams, session *models.Principal) middleware.Responder {
		payload, err := getTenantLogsResponse(session, params)
		if err != nil {
			return operator_api.NewGetTenantLogsDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewGetTenantLogsOK().WithPayload(payload)
	})

	api.OperatorAPISetTenantLogsHandler = operator_api.SetTenantLogsHandlerFunc(func(params operator_api.SetTenantLogsParams, session *models.Principal) middleware.Responder {
		payload, err := setTenantLogsResponse(session, params)
		if err != nil {
			return operator_api.NewSetTenantLogsDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewSetTenantLogsOK().WithPayload(payload)
	})

	// Get Tenant Logs
	api.OperatorAPIEnableTenantLoggingHandler = operator_api.EnableTenantLoggingHandlerFunc(func(params operator_api.EnableTenantLoggingParams, session *models.Principal) middleware.Responder {
		payload, err := enableTenantLoggingResponse(session, params)
		if err != nil {
			return operator_api.NewEnableTenantLoggingDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewEnableTenantLoggingOK().WithPayload(payload)
	})

	api.OperatorAPIDisableTenantLoggingHandler = operator_api.DisableTenantLoggingHandlerFunc(func(params operator_api.DisableTenantLoggingParams, session *models.Principal) middleware.Responder {
		payload, err := disableTenantLoggingResponse(session, params)
		if err != nil {
			return operator_api.NewDisableTenantLoggingDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewDisableTenantLoggingOK().WithPayload(payload)
	})

	api.OperatorAPIGetTenantPodsHandler = operator_api.GetTenantPodsHandlerFunc(func(params operator_api.GetTenantPodsParams, session *models.Principal) middleware.Responder {
		payload, err := getTenantPodsResponse(session, params)
		if err != nil {
			return operator_api.NewGetTenantPodsDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewGetTenantPodsOK().WithPayload(payload)
	})

	api.OperatorAPIGetPodLogsHandler = operator_api.GetPodLogsHandlerFunc(func(params operator_api.GetPodLogsParams, session *models.Principal) middleware.Responder {
		payload, err := getPodLogsResponse(session, params)
		if err != nil {
			return operator_api.NewGetPodLogsDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewGetPodLogsOK().WithPayload(payload)
	})

	api.OperatorAPIGetPodEventsHandler = operator_api.GetPodEventsHandlerFunc(func(params operator_api.GetPodEventsParams, session *models.Principal) middleware.Responder {
		payload, err := getPodEventsResponse(session, params)
		if err != nil {
			return operator_api.NewGetPodEventsDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewGetPodEventsOK().WithPayload(payload)
	})

	//Get tenant monitoring info
	api.OperatorAPIGetTenantMonitoringHandler = operator_api.GetTenantMonitoringHandlerFunc(func(params operator_api.GetTenantMonitoringParams, session *models.Principal) middleware.Responder {
		payload, err := getTenantMonitoringResponse(session, params)
		if err != nil {
			return operator_api.NewGetTenantMonitoringDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewGetTenantMonitoringOK().WithPayload(payload)
	})
	//Set configuration fields for Prometheus monitoring on a tenant
	api.OperatorAPISetTenantMonitoringHandler = operator_api.SetTenantMonitoringHandlerFunc(func(params operator_api.SetTenantMonitoringParams, session *models.Principal) middleware.Responder {
		_, err := setTenantMonitoringResponse(session, params)
		if err != nil {
			return operator_api.NewSetTenantMonitoringDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewSetTenantMonitoringCreated()
	})

	// Update Tenant Pools
	api.OperatorAPITenantUpdatePoolsHandler = operator_api.TenantUpdatePoolsHandlerFunc(func(params operator_api.TenantUpdatePoolsParams, session *models.Principal) middleware.Responder {
		resp, err := getTenantUpdatePoolResponse(session, params)
		if err != nil {
			return operator_api.NewTenantUpdatePoolsDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewTenantUpdatePoolsOK().WithPayload(resp)
	})

	// Update Tenant Certificates
	api.OperatorAPITenantUpdateCertificateHandler = operator_api.TenantUpdateCertificateHandlerFunc(func(params operator_api.TenantUpdateCertificateParams, session *models.Principal) middleware.Responder {
		err := getTenantUpdateCertificatesResponse(session, params)
		if err != nil {
			return operator_api.NewTenantUpdateCertificateDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewTenantUpdateCertificateCreated()
	})

	// Update Tenant Encryption Configuration
	api.OperatorAPITenantUpdateEncryptionHandler = operator_api.TenantUpdateEncryptionHandlerFunc(func(params operator_api.TenantUpdateEncryptionParams, session *models.Principal) middleware.Responder {
		err := getTenantUpdateEncryptionResponse(session, params)
		if err != nil {
			return operator_api.NewTenantUpdateEncryptionDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewTenantUpdateEncryptionCreated()
	})

	// Delete tenant Encryption Configuration
	api.OperatorAPITenantDeleteEncryptionHandler = operator_api.TenantDeleteEncryptionHandlerFunc(func(params operator_api.TenantDeleteEncryptionParams, session *models.Principal) middleware.Responder {
		err := getTenantDeleteEncryptionResponse(session, params)
		if err != nil {
			return operator_api.NewTenantDeleteEncryptionDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewTenantDeleteEncryptionNoContent()
	})

	// Get Tenant Encryption Configuration
	api.OperatorAPITenantEncryptionInfoHandler = operator_api.TenantEncryptionInfoHandlerFunc(func(params operator_api.TenantEncryptionInfoParams, session *models.Principal) middleware.Responder {
		configuration, err := getTenantEncryptionInfoResponse(session, params)
		if err != nil {
			return operator_api.NewTenantEncryptionInfoDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewTenantEncryptionInfoOK().WithPayload(configuration)
	})

	// Get Tenant YAML
	api.OperatorAPIGetTenantYAMLHandler = operator_api.GetTenantYAMLHandlerFunc(func(params operator_api.GetTenantYAMLParams, principal *models.Principal) middleware.Responder {
		payload, err := getTenantYAML(principal, params)
		if err != nil {
			return operator_api.NewGetTenantYAMLDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewGetTenantYAMLOK().WithPayload(payload)
	})
	// Update Tenant YAML
	api.OperatorAPIPutTenantYAMLHandler = operator_api.PutTenantYAMLHandlerFunc(func(params operator_api.PutTenantYAMLParams, principal *models.Principal) middleware.Responder {
		err := getUpdateTenantYAML(principal, params)
		if err != nil {
			return operator_api.NewPutTenantYAMLDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewPutTenantYAMLCreated()
	})
	// Get Tenant Events
	api.OperatorAPIGetTenantEventsHandler = operator_api.GetTenantEventsHandlerFunc(func(params operator_api.GetTenantEventsParams, principal *models.Principal) middleware.Responder {
		payload, err := getTenantEventsResponse(principal, params)
		if err != nil {
			return operator_api.NewGetTenantEventsDefault(int(err.Code)).WithPayload(err)
		}
		return operator_api.NewGetTenantEventsOK().WithPayload(payload)
	})
}

// getDeleteTenantResponse gets the output of deleting a minio instance
func getDeleteTenantResponse(session *models.Principal, params operator_api.DeleteTenantParams) *models.Error {
	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return prepareError(err)
	}
	// get Kubernetes Client
	clientset, err := cluster.K8sClient(session.STSSessionToken)
	if err != nil {
		return prepareError(err)
	}
	opClient := &operatorClient{
		client: opClientClientSet,
	}
	deleteTenantPVCs := false
	if params.Body != nil {
		deleteTenantPVCs = params.Body.DeletePvcs
	}

	tenant, err := opClient.TenantGet(params.HTTPRequest.Context(), params.Namespace, params.Tenant, metav1.GetOptions{})
	if err != nil {
		return prepareError(err)
	}
	tenant.EnsureDefaults()

	if err = deleteTenantAction(params.HTTPRequest.Context(), opClient, clientset.CoreV1(), tenant, deleteTenantPVCs); err != nil {
		return prepareError(err)
	}
	return nil
}

// deleteTenantAction performs the actions of deleting a tenant
//
// It also adds the option of deleting the tenant's underlying pvcs if deletePvcs set
func deleteTenantAction(
	ctx context.Context,
	operatorClient OperatorClientI,
	clientset v1.CoreV1Interface,
	tenant *miniov2.Tenant,
	deletePvcs bool) error {

	err := operatorClient.TenantDelete(ctx, tenant.Namespace, tenant.Name, metav1.DeleteOptions{})
	if err != nil {
		// try to delete pvc even if the tenant doesn't exist anymore but only if deletePvcs is set to true,
		// else, we return the error
		if (deletePvcs && !k8sErrors.IsNotFound(err)) || !deletePvcs {
			return err
		}
	}

	if deletePvcs {

		// delete MinIO PVCs
		opts := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", miniov2.TenantLabel, tenant.Name),
		}
		err = clientset.PersistentVolumeClaims(tenant.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, opts)
		if err != nil {
			return err
		}
		// delete postgres PVCs

		logOpts := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", miniov2.LogDBInstanceLabel, tenant.LogStatefulsetName()),
		}
		err := clientset.PersistentVolumeClaims(tenant.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, logOpts)
		if err != nil {
			return err
		}

		// delete prometheus PVCs

		promOpts := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", miniov2.PrometheusInstanceLabel, tenant.PrometheusStatefulsetName()),
		}

		if err := clientset.PersistentVolumeClaims(tenant.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, promOpts); err != nil {
			return err
		}

		// delete all tenant's secrets only if deletePvcs = true
		return clientset.Secrets(tenant.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, opts)
	}
	return nil
}

// getDeleteTenantResponse gets the output of deleting a minio instance
func getDeletePodResponse(session *models.Principal, params operator_api.DeletePodParams) *models.Error {
	ctx := context.Background()
	// get Kubernetes Client
	clientset, err := cluster.K8sClient(session.STSSessionToken)
	if err != nil {
		return prepareError(err)
	}
	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("v1.min.io/tenant=%s", params.Tenant),
		FieldSelector: fmt.Sprintf("metadata.name=%s%s", params.Tenant, params.PodName[len(params.Tenant):]),
	}
	if err = clientset.CoreV1().Pods(params.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOpts); err != nil {
		return prepareError(err)
	}
	return nil
}

// GetTenantServiceURL gets tenant's service url with the proper scheme and port
func GetTenantServiceURL(mi *miniov2.Tenant) (svcURL string) {
	scheme := "http"
	port := miniov2.MinIOPortLoadBalancerSVC
	if mi.AutoCert() || mi.ExternalCert() {
		scheme = "https"
		port = miniov2.MinIOTLSPortLoadBalancerSVC
	}
	return fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(mi.MinIOFQDNServiceName(), strconv.Itoa(port)))
}

func getTenantAdminClient(ctx context.Context, client K8sClientI, tenant *miniov2.Tenant, svcURL string) (*madmin.AdminClient, error) {
	tenantCreds, err := getTenantCreds(ctx, client, tenant)
	if err != nil {
		return nil, err
	}
	sessionToken := ""
	mAdmin, pErr := restapi.NewAdminClientWithInsecure(svcURL, tenantCreds.accessKey, tenantCreds.secretKey, sessionToken, true)
	if pErr != nil {
		return nil, pErr.Cause
	}
	return mAdmin, nil
}

type tenantKeys struct {
	accessKey string
	secretKey string
}

func getTenantCreds(ctx context.Context, client K8sClientI, tenant *miniov2.Tenant) (*tenantKeys, error) {
	tenantConfiguration, err := GetTenantConfiguration(ctx, client, tenant)
	if err != nil {
		return nil, err
	}
	tenantAccessKey, ok := tenantConfiguration["accesskey"]
	if !ok {
		restapi.LogError("tenant's secret doesn't contain accesskey")
		return nil, restapi.ErrorGeneric
	}
	tenantSecretKey, ok := tenantConfiguration["secretkey"]
	if !ok {
		restapi.LogError("tenant's secret doesn't contain secretkey")
		return nil, restapi.ErrorGeneric
	}
	return &tenantKeys{accessKey: tenantAccessKey, secretKey: tenantSecretKey}, nil
}

func getTenant(ctx context.Context, operatorClient OperatorClientI, namespace, tenantName string) (*miniov2.Tenant, error) {
	tenant, err := operatorClient.TenantGet(ctx, namespace, tenantName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return tenant, nil
}

func isPrometheusEnabled(annotations map[string]string) bool {
	if annotations == nil {
		return false
	}
	// if one of the following prometheus annotations are not present
	// we consider the tenant as not integrated with prometheus
	if _, ok := annotations[prometheusPath]; !ok {
		return false
	}
	if _, ok := annotations[prometheusPort]; !ok {
		return false
	}
	if _, ok := annotations[prometheusScrape]; !ok {
		return false
	}
	return true
}

func getTenantInfo(tenant *miniov2.Tenant) *models.Tenant {
	var pools []*models.Pool
	var totalSize int64
	for _, p := range tenant.Spec.Pools {
		pools = append(pools, parseTenantPool(&p))
		poolSize := int64(p.Servers) * int64(p.VolumesPerServer) * p.VolumeClaimTemplate.Spec.Resources.Requests.Storage().Value()
		totalSize = totalSize + poolSize
	}
	var deletion string
	if tenant.ObjectMeta.DeletionTimestamp != nil {
		deletion = tenant.ObjectMeta.DeletionTimestamp.Format(time.RFC3339)
	}

	return &models.Tenant{
		CreationDate:     tenant.ObjectMeta.CreationTimestamp.Format(time.RFC3339),
		DeletionDate:     deletion,
		Name:             tenant.Name,
		TotalSize:        totalSize,
		CurrentState:     tenant.Status.CurrentState,
		Pools:            pools,
		Namespace:        tenant.ObjectMeta.Namespace,
		Image:            tenant.Spec.Image,
		EnablePrometheus: isPrometheusEnabled(tenant.Annotations),
	}
}

func parseCertificate(name string, rawCert []byte) (*models.CertificateInfo, error) {
	block, _ := pem.Decode(rawCert)
	if block == nil {
		return nil, errors.New("certificate failed to decode")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	domains := []string{}
	// append certificate domain names
	if len(cert.DNSNames) > 0 {
		domains = append(domains, cert.DNSNames...)
	}
	// append certificate IPs
	if len(cert.IPAddresses) > 0 {
		for _, ip := range cert.IPAddresses {
			domains = append(domains, ip.String())
		}
	}
	return &models.CertificateInfo{
		SerialNumber: cert.SerialNumber.String(),
		Name:         name,
		Domains:      domains,
		Expiry:       cert.NotAfter.Format(time.RFC3339),
	}, nil
}

// parseTenantCertificates convert public key pem certificates stored in k8s secrets for a given Tenant into x509 certificates
func parseTenantCertificates(ctx context.Context, clientSet K8sClientI, namespace string, secrets []*miniov2.LocalCertificateReference) ([]*models.CertificateInfo, error) {
	var certificates []*models.CertificateInfo
	publicKey := "public.crt"
	// Iterate over TLS secrets and build array of CertificateInfo structure
	// that will be used to display information about certs in the UI
	for _, secret := range secrets {
		keyPair, err := clientSet.getSecret(ctx, namespace, secret.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if secret.Type == "kubernetes.io/tls" || secret.Type == "cert-manager.io/v1alpha2" {
			publicKey = "tls.crt"
		}
		// Extract public key from certificate TLS secret
		if rawCert, ok := keyPair.Data[publicKey]; ok {
			block, _ := pem.Decode(rawCert)
			if block == nil {
				// If certificate failed to decode skip
				continue
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			domains := []string{}
			// append certificate domain names
			if len(cert.DNSNames) > 0 {
				domains = append(domains, cert.DNSNames...)
			}
			// append certificate IPs
			if len(cert.IPAddresses) > 0 {
				for _, ip := range cert.IPAddresses {
					domains = append(domains, ip.String())
				}
			}
			certificates = append(certificates, &models.CertificateInfo{
				SerialNumber: cert.SerialNumber.String(),
				Name:         secret.Name,
				Domains:      domains,
				Expiry:       cert.NotAfter.Format(time.RFC3339),
			})
		}
	}
	return certificates, nil
}

func getTenantSecurity(ctx context.Context, clientSet K8sClientI, tenant *miniov2.Tenant) (response *models.TenantSecurityResponse, err error) {
	var minioExternalCertificates []*models.CertificateInfo
	var minioExternalCaCertificates []*models.CertificateInfo
	// Certificates used by MinIO server
	if minioExternalCertificates, err = parseTenantCertificates(ctx, clientSet, tenant.Namespace, tenant.Spec.ExternalCertSecret); err != nil {
		return nil, err
	}
	// CA Certificates used by MinIO server
	if minioExternalCaCertificates, err = parseTenantCertificates(ctx, clientSet, tenant.Namespace, tenant.Spec.ExternalCaCertSecret); err != nil {
		return nil, err
	}
	return &models.TenantSecurityResponse{
		AutoCert: tenant.AutoCert(),
		CustomCertificates: &models.TenantSecurityResponseCustomCertificates{
			Minio:    minioExternalCertificates,
			MinioCAs: minioExternalCaCertificates,
		},
	}, nil
}

func getTenantSecurityResponse(session *models.Principal, params operator_api.TenantSecurityParams) (*models.TenantSecurityResponse, *models.Error) {
	// 5 seconds timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return nil, prepareError(err)
	}
	opClient := &operatorClient{
		client: opClientClientSet,
	}
	minTenant, err := getTenant(ctx, opClient, params.Namespace, params.Tenant)
	if err != nil {
		return nil, prepareError(err)
	}
	// get Kubernetes Client
	clientSet, err := cluster.K8sClient(session.STSSessionToken)
	k8sClient := k8sClient{
		client: clientSet,
	}
	if err != nil {
		return nil, prepareError(err)
	}
	info, err := getTenantSecurity(ctx, &k8sClient, minTenant)
	if err != nil {
		return nil, prepareError(err)
	}
	return info, nil
}

func getUpdateTenantSecurityResponse(session *models.Principal, params operator_api.UpdateTenantSecurityParams) *models.Error {
	// 5 seconds timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return prepareError(err)
	}
	// get Kubernetes Client
	clientSet, err := cluster.K8sClient(session.STSSessionToken)
	if err != nil {
		return prepareError(err)
	}
	k8sClient := k8sClient{
		client: clientSet,
	}
	opClient := &operatorClient{
		client: opClientClientSet,
	}
	if err := updateTenantSecurity(ctx, opClient, &k8sClient, params.Namespace, params); err != nil {
		return prepareError(err, errors.New("unable to update tenant"))
	}
	return nil
}

// updateTenantSecurity
func updateTenantSecurity(ctx context.Context, operatorClient OperatorClientI, client K8sClientI, namespace string, params operator_api.UpdateTenantSecurityParams) error {
	minInst, err := operatorClient.TenantGet(ctx, namespace, params.Tenant, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// Update AutoCert
	minInst.Spec.RequestAutoCert = &params.Body.AutoCert
	var newMinIOExternalCertSecret []*miniov2.LocalCertificateReference
	var newMinIOExternalCaCertSecret []*miniov2.LocalCertificateReference
	// Remove Certificate Secrets from MinIO (Tenant.Spec.ExternalCertSecret)
	for _, certificate := range minInst.Spec.ExternalCertSecret {
		skip := false
		for _, certificateToBeDeleted := range params.Body.CustomCertificates.SecretsToBeDeleted {
			if certificate.Name == certificateToBeDeleted {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		newMinIOExternalCertSecret = append(newMinIOExternalCertSecret, certificate)
	}
	// Remove Certificate Secrets from MinIO CAs (Tenant.Spec.ExternalCaCertSecret)
	for _, certificate := range minInst.Spec.ExternalCaCertSecret {
		skip := false
		for _, certificateToBeDeleted := range params.Body.CustomCertificates.SecretsToBeDeleted {
			if certificate.Name == certificateToBeDeleted {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		newMinIOExternalCaCertSecret = append(newMinIOExternalCaCertSecret, certificate)
	}
	//Create new Certificate Secrets for MinIO
	secretName := fmt.Sprintf("%s-%s", minInst.Name, strings.ToLower(utils.RandomCharString(5)))
	externalCertSecretName := fmt.Sprintf("%s-external-certificates", secretName)
	externalCertSecrets, err := createOrReplaceExternalCertSecrets(ctx, client, minInst.Namespace, params.Body.CustomCertificates.Minio, externalCertSecretName, minInst.Name)
	if err != nil {
		return err
	}
	newMinIOExternalCertSecret = append(newMinIOExternalCertSecret, externalCertSecrets...)
	// Create new CAs Certificate Secrets for MinIO
	var caCertificates []tenantSecret
	for i, caCertificate := range params.Body.CustomCertificates.MinioCAs {
		certificateContent, err := base64.StdEncoding.DecodeString(caCertificate)
		if err != nil {
			return err
		}
		caCertificates = append(caCertificates, tenantSecret{
			Name: fmt.Sprintf("%s-ca-certificate-%d", secretName, i),
			Content: map[string][]byte{
				"public.crt": certificateContent,
			},
		})
	}
	if len(caCertificates) > 0 {
		certificateSecrets, err := createOrReplaceSecrets(ctx, client, minInst.Namespace, caCertificates, minInst.Name)
		if err != nil {
			return err
		}
		newMinIOExternalCaCertSecret = append(newMinIOExternalCaCertSecret, certificateSecrets...)
	}
	// Update External Certificates
	minInst.Spec.ExternalCertSecret = newMinIOExternalCertSecret
	minInst.Spec.ExternalCaCertSecret = newMinIOExternalCaCertSecret
	_, err = operatorClient.TenantUpdate(ctx, minInst, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	// Remove Certificate Secrets from Tenant namespace
	for _, secretName := range params.Body.CustomCertificates.SecretsToBeDeleted {
		err = client.deleteSecret(ctx, minInst.Namespace, secretName, metav1.DeleteOptions{})
		if err != nil {
			restapi.LogError("error deleting secret: %v", err)
		}
	}
	return nil
}

func listTenants(ctx context.Context, operatorClient OperatorClientI, namespace string, limit *int32) (*models.ListTenantsResponse, error) {
	listOpts := metav1.ListOptions{
		Limit: 10,
	}

	if limit != nil {
		listOpts.Limit = int64(*limit)
	}

	minTenants, err := operatorClient.TenantList(ctx, namespace, listOpts)
	if err != nil {
		return nil, err
	}

	var tenants []*models.TenantList

	for _, tenant := range minTenants.Items {
		var totalSize int64
		var instanceCount int64
		var volumeCount int64
		for _, pool := range tenant.Spec.Pools {
			instanceCount = instanceCount + int64(pool.Servers)
			volumeCount = volumeCount + int64(pool.Servers*pool.VolumesPerServer)
			if pool.VolumeClaimTemplate != nil {
				poolSize := int64(pool.VolumesPerServer) * int64(pool.Servers) * pool.VolumeClaimTemplate.Spec.Resources.Requests.Storage().Value()
				totalSize = totalSize + poolSize
			}
		}

		var deletion string
		if tenant.ObjectMeta.DeletionTimestamp != nil {
			deletion = tenant.ObjectMeta.DeletionTimestamp.Format(time.RFC3339)
		}

		tenants = append(tenants, &models.TenantList{
			CreationDate:     tenant.ObjectMeta.CreationTimestamp.Format(time.RFC3339),
			DeletionDate:     deletion,
			Name:             tenant.ObjectMeta.Name,
			PoolCount:        int64(len(tenant.Spec.Pools)),
			InstanceCount:    instanceCount,
			VolumeCount:      volumeCount,
			CurrentState:     tenant.Status.CurrentState,
			Namespace:        tenant.ObjectMeta.Namespace,
			TotalSize:        totalSize,
			HealthStatus:     string(tenant.Status.HealthStatus),
			CapacityRaw:      tenant.Status.Usage.RawCapacity,
			CapacityRawUsage: tenant.Status.Usage.RawUsage,
			Capacity:         tenant.Status.Usage.Capacity,
			CapacityUsage:    tenant.Status.Usage.Usage,
		})
	}

	return &models.ListTenantsResponse{
		Tenants: tenants,
		Total:   int64(len(tenants)),
	}, nil
}

func getListAllTenantsResponse(session *models.Principal, params operator_api.ListAllTenantsParams) (*models.ListTenantsResponse, *models.Error) {
	ctx := context.Background()
	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return nil, prepareError(err)
	}
	opClient := &operatorClient{
		client: opClientClientSet,
	}
	listT, err := listTenants(ctx, opClient, "", params.Limit)
	if err != nil {
		return nil, prepareError(err)
	}
	return listT, nil
}

// getListTenantsResponse list tenants by namespace
func getListTenantsResponse(session *models.Principal, params operator_api.ListTenantsParams) (*models.ListTenantsResponse, *models.Error) {
	ctx := context.Background()
	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return nil, prepareError(err)
	}
	opClient := &operatorClient{
		client: opClientClientSet,
	}
	listT, err := listTenants(ctx, opClient, params.Namespace, params.Limit)
	if err != nil {
		return nil, prepareError(err)
	}
	return listT, nil
}

// setImageRegistry creates a secret to store the private registry credentials, if one exist it updates the existing one
// returns the name of the secret created/updated
func setImageRegistry(ctx context.Context, req *models.ImageRegistry, clientset v1.CoreV1Interface, namespace, tenantName string) (string, error) {
	if req == nil || req.Registry == nil || req.Username == nil || req.Password == nil {
		return "", nil
	}

	credentials := make(map[string]imageRegistryCredentials)
	// username:password encoded
	authData := []byte(fmt.Sprintf("%s:%s", *req.Username, *req.Password))
	authStr := base64.StdEncoding.EncodeToString(authData)

	credentials[*req.Registry] = imageRegistryCredentials{
		Username: *req.Username,
		Password: *req.Password,
		Auth:     authStr,
	}
	imRegistry := imageRegistry{
		Auths: credentials,
	}
	imRegistryJSON, err := json.Marshal(imRegistry)
	if err != nil {
		return "", err
	}

	pullSecretName := fmt.Sprintf("%s-regcred", tenantName)
	secretCredentials := map[string][]byte{
		corev1.DockerConfigJsonKey: []byte(string(imRegistryJSON)),
	}
	// Get or Create secret if it doesn't exist
	currentSecret, err := clientset.Secrets(namespace).Get(ctx, pullSecretName, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			instanceSecret := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: pullSecretName,
					Labels: map[string]string{
						miniov2.TenantLabel: tenantName,
					},
				},
				Data: secretCredentials,
				Type: corev1.SecretTypeDockerConfigJson,
			}
			_, err = clientset.Secrets(namespace).Create(ctx, &instanceSecret, metav1.CreateOptions{})
			if err != nil {
				return "", err
			}
			return pullSecretName, nil
		}
		return "", err
	}
	currentSecret.Data = secretCredentials
	_, err = clientset.Secrets(namespace).Update(ctx, currentSecret, metav1.UpdateOptions{})
	if err != nil {
		return "", err
	}
	return pullSecretName, nil
}

// addAnnotations will merge two annotation maps
func addAnnotations(annotationsOne, annotationsTwo map[string]string) map[string]string {
	if annotationsOne == nil {
		annotationsOne = map[string]string{}
	}
	for key, value := range annotationsTwo {
		annotationsOne[key] = value
	}
	return annotationsOne
}

// removeAnnotations will remove keys from the first annotations map based on the second one
func removeAnnotations(annotationsOne, annotationsTwo map[string]string) map[string]string {
	if annotationsOne == nil {
		annotationsOne = map[string]string{}
	}
	for key := range annotationsTwo {
		delete(annotationsOne, key)
	}
	return annotationsOne
}

func getUpdateTenantResponse(session *models.Principal, params operator_api.UpdateTenantParams) *models.Error {
	ctx := context.Background()
	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return prepareError(err)
	}
	// get Kubernetes Client
	clientSet, err := cluster.K8sClient(session.STSSessionToken)
	if err != nil {
		return prepareError(err)
	}
	opClient := &operatorClient{
		client: opClientClientSet,
	}
	httpC := &utils2.HTTPClient{
		Client: &http.Client{
			Timeout: 4 * time.Second,
		},
	}
	if err := updateTenantAction(ctx, opClient, clientSet.CoreV1(), httpC, params.Namespace, params); err != nil {
		return prepareError(err, errors.New("unable to update tenant"))
	}
	return nil
}

// addTenantPool creates a pool to a defined tenant
func addTenantPool(ctx context.Context, operatorClient OperatorClientI, params operator_api.TenantAddPoolParams) error {
	tenant, err := operatorClient.TenantGet(ctx, params.Namespace, params.Tenant, metav1.GetOptions{})
	if err != nil {
		return err
	}

	poolParams := params.Body
	pool, err := parseTenantPoolRequest(poolParams)
	if err != nil {
		return err
	}
	tenant.Spec.Pools = append(tenant.Spec.Pools, *pool)
	payloadBytes, err := json.Marshal(tenant)
	if err != nil {
		return err
	}

	_, err = operatorClient.TenantPatch(ctx, params.Namespace, tenant.Name, types.MergePatchType, payloadBytes, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}

func getTenantAddPoolResponse(session *models.Principal, params operator_api.TenantAddPoolParams) *models.Error {
	ctx := context.Background()
	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return prepareError(err)
	}
	opClient := &operatorClient{
		client: opClientClientSet,
	}
	if err := addTenantPool(ctx, opClient, params); err != nil {
		return prepareError(err, errors.New("unable to add pool"))
	}
	return nil
}

// getTenantUsageResponse returns the usage of a tenant
func getTenantUsageResponse(session *models.Principal, params operator_api.GetTenantUsageParams) (*models.TenantUsage, *models.Error) {
	// 30 seconds timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return nil, prepareError(err, errorUnableToGetTenantUsage)
	}
	clientSet, err := cluster.K8sClient(session.STSSessionToken)
	if err != nil {
		return nil, prepareError(err, errorUnableToGetTenantUsage)
	}

	opClient := &operatorClient{
		client: opClientClientSet,
	}
	k8sClient := &k8sClient{
		client: clientSet,
	}

	minTenant, err := getTenant(ctx, opClient, params.Namespace, params.Tenant)
	if err != nil {
		return nil, prepareError(err, errorUnableToGetTenantUsage)
	}
	minTenant.EnsureDefaults()

	svcURL := GetTenantServiceURL(minTenant)
	// getTenantAdminClient will use all certificates under ~/.console/certs/CAs to trust the TLS connections with MinIO tenants
	mAdmin, err := getTenantAdminClient(
		ctx,
		k8sClient,
		minTenant,
		svcURL,
	)
	if err != nil {
		return nil, prepareError(err, errorUnableToGetTenantUsage)
	}
	// create a minioClient interface implementation
	// defining the client to be used
	adminClient := restapi.AdminClient{Client: mAdmin}
	// serialize output
	adminInfo, err := restapi.GetAdminInfo(ctx, adminClient)
	if err != nil {
		return nil, prepareError(err, errorUnableToGetTenantUsage)
	}
	info := &models.TenantUsage{Used: adminInfo.Usage, DiskUsed: adminInfo.DisksUsage}
	return info, nil
}

// getTenantLogsResponse returns the logs of a tenant
func getTenantLogsResponse(session *models.Principal, params operator_api.GetTenantLogsParams) (*models.TenantLogs, *models.Error) {
	// 30 seconds timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return nil, prepareError(err, errorUnableToGetTenantLogs)
	}

	opClient := &operatorClient{
		client: opClientClientSet,
	}

	minTenant, err := getTenant(ctx, opClient, params.Namespace, params.Tenant)
	if err != nil {
		return nil, prepareError(err, errorUnableToGetTenantLogs)
	}
	if minTenant.Spec.Log == nil {
		retval := &models.TenantLogs{
			Disabled: true,
		}
		return retval, nil
	}
	annotations := []*models.Annotation{}
	for k, v := range minTenant.Spec.Log.Annotations {
		annotations = append(annotations, &models.Annotation{Key: k, Value: v})
	}
	labels := []*models.Label{}
	for k, v := range minTenant.Spec.Log.Labels {
		labels = append(labels, &models.Label{Key: k, Value: v})
	}
	nodeSelector := []*models.NodeSelector{}
	for k, v := range minTenant.Spec.Log.NodeSelector {
		nodeSelector = append(nodeSelector, &models.NodeSelector{Key: k, Value: v})
	}

	if minTenant.Spec.Log.Db == nil {
		minTenant.Spec.Log.Db = &miniov2.LogDbConfig{}
	}

	dbAnnotations := []*models.Annotation{}
	for k, v := range minTenant.Spec.Log.Db.Annotations {
		dbAnnotations = append(dbAnnotations, &models.Annotation{Key: k, Value: v})
	}
	dbLabels := []*models.Label{}
	for k, v := range minTenant.Spec.Log.Db.Labels {
		dbLabels = append(dbLabels, &models.Label{Key: k, Value: v})
	}
	dbNodeSelector := []*models.NodeSelector{}
	for k, v := range minTenant.Spec.Log.Db.NodeSelector {
		dbNodeSelector = append(dbNodeSelector, &models.NodeSelector{Key: k, Value: v})
	}

	if minTenant.Spec.Log.Audit == nil || minTenant.Spec.Log.Audit.DiskCapacityGB == nil {
		minTenant.Spec.Log.Audit = &miniov2.AuditConfig{DiskCapacityGB: swag.Int(0)}
	}

	retval := &models.TenantLogs{
		Image:                minTenant.Spec.Log.Image,
		DiskCapacityGB:       fmt.Sprintf("%d", *minTenant.Spec.Log.Audit.DiskCapacityGB),
		Annotations:          annotations,
		Labels:               labels,
		NodeSelector:         nodeSelector,
		ServiceAccountName:   minTenant.Spec.Log.ServiceAccountName,
		DbImage:              minTenant.Spec.Log.Db.Image,
		DbAnnotations:        dbAnnotations,
		DbLabels:             dbLabels,
		DbNodeSelector:       dbNodeSelector,
		DbServiceAccountName: minTenant.Spec.Log.Db.ServiceAccountName,
		Disabled:             false,
	}

	var requestedCPU string
	var requestedMem string
	var requestedDBCPU string
	var requestedDBMem string
	if minTenant.Spec.Log.Resources.Requests != nil {
		requestedCPUQ := minTenant.Spec.Log.Resources.Requests["cpu"]
		requestedCPU = strconv.FormatInt(requestedCPUQ.Value(), 10)
		requestedMemQ := minTenant.Spec.Log.Resources.Requests["memory"]
		requestedMem = strconv.FormatInt(requestedMemQ.Value(), 10)

		requestedDBCPUQ := minTenant.Spec.Log.Db.Resources.Requests["cpu"]
		requestedDBCPU = strconv.FormatInt(requestedDBCPUQ.Value(), 10)
		requestedDBMemQ := minTenant.Spec.Log.Db.Resources.Requests["memory"]
		requestedDBMem = strconv.FormatInt(requestedDBMemQ.Value(), 10)

		retval.LogCPURequest = requestedCPU
		retval.LogMemRequest = requestedMem
		retval.LogDBCPURequest = requestedDBCPU
		retval.LogDBMemRequest = requestedDBMem
	}
	return retval, nil
}

// setTenantLogsResponse returns the logs of a tenant
func setTenantLogsResponse(session *models.Principal, params operator_api.SetTenantLogsParams) (bool, *models.Error) {

	// 30 seconds timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return false, prepareError(err, errorUnableToGetTenantUsage)
	}

	opClient := &operatorClient{
		client: opClientClientSet,
	}

	minTenant, err := getTenant(ctx, opClient, params.Namespace, params.Tenant)
	if err != nil {
		return false, prepareError(err, errorUnableToGetTenantUsage)
	}

	var labels = make(map[string]string)
	for i := 0; i < len(params.Data.Labels); i++ {
		if params.Data.Labels[i] != nil {
			labels[params.Data.Labels[i].Key] = params.Data.Labels[i].Value
		}
	}
	minTenant.Spec.Log.Labels = labels
	var annotations = make(map[string]string)
	for i := 0; i < len(params.Data.Annotations); i++ {
		if params.Data.Annotations[i] != nil {
			annotations[params.Data.Annotations[i].Key] = params.Data.Annotations[i].Value
		}
	}
	minTenant.Spec.Log.Annotations = annotations
	var nodeSelector = make(map[string]string)
	for i := 0; i < len(params.Data.NodeSelector); i++ {
		if params.Data.NodeSelector[i] != nil {
			nodeSelector[params.Data.NodeSelector[i].Key] = params.Data.NodeSelector[i].Value
		}
	}
	minTenant.Spec.Log.NodeSelector = nodeSelector
	logResourceRequest := make(corev1.ResourceList)

	if reflect.TypeOf(params.Data.LogCPURequest).Kind() == reflect.String && params.Data.LogCPURequest != "0Gi" && params.Data.LogCPURequest != "" {
		cpuQuantity, err := resource.ParseQuantity(params.Data.LogCPURequest)
		if err != nil {
			return false, prepareError(err)
		}
		logResourceRequest["cpu"] = cpuQuantity
		minTenant.Spec.Log.Resources.Requests = logResourceRequest
	}
	if reflect.TypeOf(params.Data.LogMemRequest).Kind() == reflect.String {
		memQuantity, err := resource.ParseQuantity(params.Data.LogMemRequest)
		if err != nil {
			return false, prepareError(err)
		}

		logResourceRequest["memory"] = memQuantity
		minTenant.Spec.Log.Resources.Requests = logResourceRequest
	}

	modified := false
	if minTenant.Spec.Log.Db != nil {
		modified = true
	}
	var dbLabels = make(map[string]string)
	for i := 0; i < len(params.Data.DbLabels); i++ {
		if params.Data.DbLabels[i] != nil {
			dbLabels[params.Data.DbLabels[i].Key] = params.Data.DbLabels[i].Value
		}
		modified = true
	}
	var dbAnnotations = make(map[string]string)
	for i := 0; i < len(params.Data.DbAnnotations); i++ {
		if params.Data.DbAnnotations[i] != nil {
			dbAnnotations[params.Data.DbAnnotations[i].Key] = params.Data.DbAnnotations[i].Value
		}
		modified = true
	}
	var dbNodeSelector = make(map[string]string)
	for i := 0; i < len(params.Data.DbNodeSelector); i++ {
		if params.Data.DbNodeSelector[i] != nil {
			dbNodeSelector[params.Data.DbNodeSelector[i].Key] = params.Data.DbNodeSelector[i].Value
		}
		modified = true
	}

	logDBResourceRequest := make(corev1.ResourceList)
	if reflect.TypeOf(params.Data.LogDBCPURequest).Kind() == reflect.String && params.Data.LogDBCPURequest != "0Gi" && params.Data.LogDBCPURequest != "" {
		dbCPUQuantity, err := resource.ParseQuantity(params.Data.LogDBCPURequest)
		if err != nil {
			return false, prepareError(err)
		}
		logDBResourceRequest["cpu"] = dbCPUQuantity
		minTenant.Spec.Log.Db.Resources.Requests = logDBResourceRequest
	}
	if reflect.TypeOf(params.Data.LogDBMemRequest).Kind() == reflect.String {
		dbMemQuantity, err := resource.ParseQuantity(params.Data.LogDBMemRequest)
		if err != nil {
			return false, prepareError(err)
		}
		logDBResourceRequest["memory"] = dbMemQuantity
		minTenant.Spec.Log.Db.Resources.Requests = logDBResourceRequest
	}
	minTenant.Spec.Log.Image = params.Data.Image
	diskCapacityGB, err := strconv.Atoi(params.Data.DiskCapacityGB)
	if err == nil {
		if minTenant.Spec.Log.Audit != nil && minTenant.Spec.Log.Audit.DiskCapacityGB != nil {
			*minTenant.Spec.Log.Audit.DiskCapacityGB = diskCapacityGB
		} else {
			minTenant.Spec.Log.Audit = &miniov2.AuditConfig{DiskCapacityGB: swag.Int(diskCapacityGB)}
		}
	}
	minTenant.Spec.Log.ServiceAccountName = params.Data.ServiceAccountName
	if params.Data.DbImage != "" || params.Data.DbServiceAccountName != "" {
		modified = true
	}
	if modified {
		if minTenant.Spec.Log.Db == nil {
			//Default class name for Log search
			diskSpaceFromAPI := int64(5) * humanize.GiByte // Default is 5Gi
			logSearchStorageClass := "standard"

			logSearchDiskSpace := resource.NewQuantity(diskSpaceFromAPI, resource.DecimalExponent)

			minTenant.Spec.Log.Db = &miniov2.LogDbConfig{
				VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name: params.Tenant + "-log",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: *logSearchDiskSpace,
							},
						},
						StorageClassName: &logSearchStorageClass,
					},
				},
				Labels:             dbLabels,
				Annotations:        dbAnnotations,
				NodeSelector:       dbNodeSelector,
				Image:              params.Data.DbImage,
				ServiceAccountName: params.Data.DbServiceAccountName,
				Resources: corev1.ResourceRequirements{
					Requests: minTenant.Spec.Log.Db.Resources.Requests,
				},
			}
		} else {
			minTenant.Spec.Log.Db.Labels = dbLabels
			minTenant.Spec.Log.Db.Annotations = dbAnnotations
			minTenant.Spec.Log.Db.NodeSelector = dbNodeSelector
			minTenant.Spec.Log.Db.Image = params.Data.DbImage
			minTenant.Spec.Log.Db.ServiceAccountName = params.Data.DbServiceAccountName
		}
	}

	_, err = opClient.TenantUpdate(ctx, minTenant, metav1.UpdateOptions{})
	if err != nil {
		return false, prepareError(err)
	}
	return true, nil
}

// enableTenantLoggingResponse enables Tenant Logging
func enableTenantLoggingResponse(session *models.Principal, params operator_api.EnableTenantLoggingParams) (bool, *models.Error) {
	// 30 seconds timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return false, prepareError(err, errorUnableToGetTenantUsage)
	}

	opClient := &operatorClient{
		client: opClientClientSet,
	}

	minTenant, err := getTenant(ctx, opClient, params.Namespace, params.Tenant)
	if err != nil {
		return false, prepareError(err, errorUnableToGetTenantUsage)
	}
	minTenant.EnsureDefaults()

	//Default class name for Log search
	diskSpaceFromAPI := int64(5) * humanize.GiByte // Default is 5Gi
	logSearchStorageClass := "standard"

	logSearchDiskSpace := resource.NewQuantity(diskSpaceFromAPI, resource.DecimalExponent)

	auditMaxCap := 10
	if (diskSpaceFromAPI / humanize.GiByte) < int64(auditMaxCap) {
		auditMaxCap = int(diskSpaceFromAPI / humanize.GiByte)
	}

	minTenant.Spec.Log = &miniov2.LogConfig{
		Audit: &miniov2.AuditConfig{DiskCapacityGB: swag.Int(auditMaxCap)},
		Db: &miniov2.LogDbConfig{
			VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: params.Tenant + "-log",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: *logSearchDiskSpace,
						},
					},
					StorageClassName: &logSearchStorageClass,
				},
			},
		},
	}

	_, err = opClient.TenantUpdate(ctx, minTenant, metav1.UpdateOptions{})
	if err != nil {
		return false, prepareError(err)
	}
	return true, nil
}

// disableTenantLoggingResponse disables Tenant Logging
func disableTenantLoggingResponse(session *models.Principal, params operator_api.DisableTenantLoggingParams) (bool, *models.Error) {
	// 30 seconds timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return false, prepareError(err, errorUnableToGetTenantUsage)
	}
	if err != nil {
		return false, prepareError(err, errorUnableToGetTenantUsage)
	}

	opClient := &operatorClient{
		client: opClientClientSet,
	}

	minTenant, err := getTenant(ctx, opClient, params.Namespace, params.Tenant)
	if err != nil {
		return false, prepareError(err, errorUnableToGetTenantUsage)
	}
	minTenant.EnsureDefaults()
	minTenant.Spec.Log = nil

	_, err = opClient.TenantUpdate(ctx, minTenant, metav1.UpdateOptions{})
	if err != nil {
		return false, prepareError(err)
	}
	return true, nil
}

func getTenantPodsResponse(session *models.Principal, params operator_api.GetTenantPodsParams) ([]*models.TenantPod, *models.Error) {
	ctx := context.Background()
	clientset, err := cluster.K8sClient(session.STSSessionToken)
	if err != nil {
		return nil, prepareError(err)
	}
	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", miniov2.TenantLabel, params.Tenant),
	}
	pods, err := clientset.CoreV1().Pods(params.Namespace).List(ctx, listOpts)
	if err != nil {
		return nil, prepareError(err)
	}
	retval := []*models.TenantPod{}
	for _, pod := range pods.Items {
		var restarts int64
		if len(pod.Status.ContainerStatuses) > 0 {
			restarts = int64(pod.Status.ContainerStatuses[0].RestartCount)
		}
		status := string(pod.Status.Phase)
		if pod.DeletionTimestamp != nil {
			status = "Terminating"
		}
		retval = append(retval, &models.TenantPod{
			Name:        swag.String(pod.Name),
			Status:      status,
			TimeCreated: pod.CreationTimestamp.Unix(),
			PodIP:       pod.Status.PodIP,
			Restarts:    restarts,
			Node:        pod.Spec.NodeName})
	}
	return retval, nil
}

func getPodLogsResponse(session *models.Principal, params operator_api.GetPodLogsParams) (string, *models.Error) {
	ctx := context.Background()
	clientset, err := cluster.K8sClient(session.STSSessionToken)
	if err != nil {
		return "", prepareError(err)
	}
	listOpts := &corev1.PodLogOptions{}
	logs := clientset.CoreV1().Pods(params.Namespace).GetLogs(params.PodName, listOpts)
	buff, err := logs.DoRaw(ctx)
	if err != nil {
		return "", prepareError(err)
	}
	return string(buff), nil
}

func getPodEventsResponse(session *models.Principal, params operator_api.GetPodEventsParams) (models.EventListWrapper, *models.Error) {
	ctx := context.Background()
	clientset, err := cluster.K8sClient(session.STSSessionToken)
	if err != nil {
		return nil, prepareError(err)
	}
	pod, err := clientset.CoreV1().Pods(params.Namespace).Get(ctx, params.PodName, metav1.GetOptions{})
	if err != nil {
		return nil, prepareError(err)
	}
	events, err := clientset.CoreV1().Events(params.Namespace).List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.uid=%s", pod.UID)})
	if err != nil {
		return nil, prepareError(err)
	}
	retval := models.EventListWrapper{}
	for i := 0; i < len(events.Items); i++ {
		retval = append(retval, &models.EventListElement{
			Namespace: events.Items[i].Namespace,
			LastSeen:  events.Items[i].LastTimestamp.Unix(),
			Message:   events.Items[i].Message,
			EventType: events.Items[i].Type,
			Reason:    events.Items[i].Reason,
		})
	}
	sort.SliceStable(retval, func(i int, j int) bool {
		return retval[i].LastSeen < retval[j].LastSeen
	})
	return retval, nil
}

//get values for prometheus metrics
func getTenantMonitoringResponse(session *models.Principal, params operator_api.GetTenantMonitoringParams) (*models.TenantMonitoringInfo, *models.Error) {
	ctx := context.Background()

	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return nil, prepareError(err)
	}

	opClient := &operatorClient{
		client: opClientClientSet,
	}

	minInst, err := opClient.TenantGet(ctx, params.Namespace, params.Tenant, metav1.GetOptions{})
	if err != nil {
		return nil, prepareError(err)
	}

	monitoringInfo := &models.TenantMonitoringInfo{}

	if minInst.Spec.Prometheus != nil {
		monitoringInfo.PrometheusEnabled = true
	} else {
		monitoringInfo.PrometheusEnabled = false
		return monitoringInfo, nil
	}

	var storageClassName string
	if minInst.Spec.Prometheus.StorageClassName != nil {
		storageClassName = *minInst.Spec.Prometheus.StorageClassName
		monitoringInfo.StorageClassName = storageClassName
	}

	var requestedCPU string
	var requestedMem string

	if minInst.Spec.Prometheus.Resources.Requests != nil {
		// Parse cpu request
		if requestedCPUQ, ok := minInst.Spec.Prometheus.Resources.Requests["cpu"]; ok && requestedCPUQ.Value() != 0 {
			requestedCPU = strconv.FormatInt(requestedCPUQ.Value(), 10)
			monitoringInfo.MonitoringCPURequest = requestedCPU
		}
		// Parse memory request
		if requestedMemQ, ok := minInst.Spec.Prometheus.Resources.Requests["memory"]; ok && requestedMemQ.Value() != 0 {
			requestedMem = strconv.FormatInt(requestedMemQ.Value(), 10)
			monitoringInfo.MonitoringMemRequest = requestedMem
		}
	}

	if len(minInst.Spec.Prometheus.Labels) != 0 && minInst.Spec.Prometheus.Labels != nil {
		mLabels := []*models.Label{}
		for k, v := range minInst.Spec.Prometheus.Labels {
			mLabels = append(mLabels, &models.Label{Key: k, Value: v})
		}
		monitoringInfo.Labels = mLabels
	}

	if len(minInst.Spec.Prometheus.Annotations) != 0 && minInst.Spec.Prometheus.Annotations != nil {
		mAnnotations := []*models.Annotation{}
		for k, v := range minInst.Spec.Prometheus.Annotations {
			mAnnotations = append(mAnnotations, &models.Annotation{Key: k, Value: v})
		}
		monitoringInfo.Annotations = mAnnotations
	}

	if len(minInst.Spec.Prometheus.NodeSelector) != 0 && minInst.Spec.Prometheus.NodeSelector != nil {
		mNodeSelector := []*models.NodeSelector{}
		for k, v := range minInst.Spec.Prometheus.NodeSelector {
			mNodeSelector = append(mNodeSelector, &models.NodeSelector{Key: k, Value: v})
		}
		monitoringInfo.NodeSelector = mNodeSelector
	}

	if *minInst.Spec.Prometheus.DiskCapacityDB != 0 {
		monitoringInfo.DiskCapacityGB = strconv.Itoa(*minInst.Spec.Prometheus.DiskCapacityDB)
	}
	if len(minInst.Spec.Prometheus.Image) != 0 {
		monitoringInfo.Image = minInst.Spec.Prometheus.Image
	}
	if len(minInst.Spec.Prometheus.InitImage) != 0 {
		monitoringInfo.InitImage = minInst.Spec.Prometheus.InitImage
	}
	if len(minInst.Spec.Prometheus.ServiceAccountName) != 0 {
		monitoringInfo.ServiceAccountName = minInst.Spec.Prometheus.ServiceAccountName
	}
	if len(minInst.Spec.Prometheus.SideCarImage) != 0 {
		monitoringInfo.SidecarImage = minInst.Spec.Prometheus.SideCarImage
	}

	return monitoringInfo, nil

}

//sets tenant Prometheus monitoring cofiguration fields to values provided
func setTenantMonitoringResponse(session *models.Principal, params operator_api.SetTenantMonitoringParams) (bool, *models.Error) {
	// 30 seconds timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return false, prepareError(err, errorUnableToGetTenantUsage)
	}

	opClient := &operatorClient{
		client: opClientClientSet,
	}

	minTenant, err := getTenant(ctx, opClient, params.Namespace, params.Tenant)
	if err != nil {
		return false, prepareError(err, errorUnableToGetTenantUsage)
	}

	if params.Data.Toggle {
		if params.Data.PrometheusEnabled {
			minTenant.Spec.Prometheus = nil
		} else {
			promDiskSpaceGB := 5
			promImage := ""
			minTenant.Spec.Prometheus = &miniov2.PrometheusConfig{
				DiskCapacityDB: swag.Int(promDiskSpaceGB),
				Image:          promImage,
			}
		}
		_, err = opClient.TenantUpdate(ctx, minTenant, metav1.UpdateOptions{})
		if err != nil {
			return false, prepareError(err)
		}
		return true, nil
	}

	var labels = make(map[string]string)
	for i := 0; i < len(params.Data.Labels); i++ {
		if params.Data.Labels[i] != nil {
			labels[params.Data.Labels[i].Key] = params.Data.Labels[i].Value
		}
	}
	var annotations = make(map[string]string)
	for i := 0; i < len(params.Data.Annotations); i++ {
		if params.Data.Annotations[i] != nil {
			annotations[params.Data.Annotations[i].Key] = params.Data.Annotations[i].Value
		}
	}
	var nodeSelector = make(map[string]string)
	for i := 0; i < len(params.Data.NodeSelector); i++ {
		if params.Data.NodeSelector[i] != nil {
			nodeSelector[params.Data.NodeSelector[i].Key] = params.Data.NodeSelector[i].Value
		}
	}

	monitoringResourceRequest := make(corev1.ResourceList)
	if params.Data.MonitoringCPURequest != "" {
		cpuQuantity, err := resource.ParseQuantity(params.Data.MonitoringCPURequest)
		if err != nil {
			return false, prepareError(err)
		}
		monitoringResourceRequest["cpu"] = cpuQuantity
	}

	if params.Data.MonitoringMemRequest != "" {
		memQuantity, err := resource.ParseQuantity(params.Data.MonitoringMemRequest)
		if err != nil {
			return false, prepareError(err)
		}
		monitoringResourceRequest["memory"] = memQuantity
	}

	minTenant.Spec.Prometheus.Resources.Requests = monitoringResourceRequest
	minTenant.Spec.Prometheus.Labels = labels
	minTenant.Spec.Prometheus.Annotations = annotations
	minTenant.Spec.Prometheus.NodeSelector = nodeSelector
	minTenant.Spec.Prometheus.Image = params.Data.Image
	minTenant.Spec.Prometheus.SideCarImage = params.Data.SidecarImage
	minTenant.Spec.Prometheus.InitImage = params.Data.InitImage
	if params.Data.StorageClassName == "" {
		minTenant.Spec.Prometheus.StorageClassName = nil
	} else {
		minTenant.Spec.Prometheus.StorageClassName = &params.Data.StorageClassName
	}

	diskCapacityGB, err := strconv.Atoi(params.Data.DiskCapacityGB)
	if err == nil {
		*minTenant.Spec.Prometheus.DiskCapacityDB = diskCapacityGB
	}
	minTenant.Spec.Prometheus.ServiceAccountName = params.Data.ServiceAccountName
	_, err = opClient.TenantUpdate(ctx, minTenant, metav1.UpdateOptions{})
	if err != nil {
		return false, prepareError(err)
	}

	return true, nil

}

// parseTenantPoolRequest parse pool request and returns the equivalent
// miniov2.Pool object
func parseTenantPoolRequest(poolParams *models.Pool) (*miniov2.Pool, error) {
	if poolParams.VolumeConfiguration == nil {
		return nil, errors.New("a volume configuration must be specified")
	}

	if poolParams.VolumeConfiguration.Size == nil || *poolParams.VolumeConfiguration.Size <= int64(0) {
		return nil, errors.New("volume size must be greater than 0")
	}

	if poolParams.Servers == nil || *poolParams.Servers <= 0 {
		return nil, errors.New("number of servers must be greater than 0")
	}

	if poolParams.VolumesPerServer == nil || *poolParams.VolumesPerServer <= 0 {
		return nil, errors.New("number of volumes per server must be greater than 0")
	}

	volumeSize := resource.NewQuantity(*poolParams.VolumeConfiguration.Size, resource.DecimalExponent)
	volTemp := corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{
			corev1.ReadWriteOnce,
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: *volumeSize,
			},
		},
	}
	if poolParams.VolumeConfiguration.StorageClassName != "" {
		volTemp.StorageClassName = &poolParams.VolumeConfiguration.StorageClassName
	}

	// parse resources' requests
	resourcesRequests := make(corev1.ResourceList)
	resourcesLimits := make(corev1.ResourceList)
	if poolParams.Resources != nil {
		for key, val := range poolParams.Resources.Requests {
			resourcesRequests[corev1.ResourceName(key)] = *resource.NewQuantity(val, resource.BinarySI)
		}
		for key, val := range poolParams.Resources.Limits {
			resourcesLimits[corev1.ResourceName(key)] = *resource.NewQuantity(val, resource.BinarySI)
		}
	}

	// parse Node Affinity
	nodeSelectorTerms := []corev1.NodeSelectorTerm{}
	preferredSchedulingTerm := []corev1.PreferredSchedulingTerm{}
	if poolParams.Affinity != nil && poolParams.Affinity.NodeAffinity != nil {
		if poolParams.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			for _, elem := range poolParams.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
				term := parseModelsNodeSelectorTerm(elem)
				nodeSelectorTerms = append(nodeSelectorTerms, term)
			}
		}
		for _, elem := range poolParams.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			pst := corev1.PreferredSchedulingTerm{
				Weight:     *elem.Weight,
				Preference: parseModelsNodeSelectorTerm(elem.Preference),
			}
			preferredSchedulingTerm = append(preferredSchedulingTerm, pst)
		}
	}
	var nodeAffinity *corev1.NodeAffinity
	if len(nodeSelectorTerms) > 0 || len(preferredSchedulingTerm) > 0 {
		nodeAffinity = &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: nodeSelectorTerms,
			},
			PreferredDuringSchedulingIgnoredDuringExecution: preferredSchedulingTerm,
		}
	}

	// parse Pod Affinity
	podAffinityTerms := []corev1.PodAffinityTerm{}
	weightedPodAffinityTerms := []corev1.WeightedPodAffinityTerm{}
	if poolParams.Affinity != nil && poolParams.Affinity.PodAffinity != nil {
		for _, elem := range poolParams.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
			podAffinityTerms = append(podAffinityTerms, parseModelPodAffinityTerm(elem))
		}
		for _, elem := range poolParams.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			wAffinityTerm := corev1.WeightedPodAffinityTerm{
				Weight:          *elem.Weight,
				PodAffinityTerm: parseModelPodAffinityTerm(elem.PodAffinityTerm),
			}
			weightedPodAffinityTerms = append(weightedPodAffinityTerms, wAffinityTerm)
		}
	}
	var podAffinity *corev1.PodAffinity
	if len(podAffinityTerms) > 0 || len(weightedPodAffinityTerms) > 0 {
		podAffinity = &corev1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution:  podAffinityTerms,
			PreferredDuringSchedulingIgnoredDuringExecution: weightedPodAffinityTerms,
		}
	}

	// parse Pod Anti Affinity
	podAntiAffinityTerms := []corev1.PodAffinityTerm{}
	weightedPodAntiAffinityTerms := []corev1.WeightedPodAffinityTerm{}
	if poolParams.Affinity != nil && poolParams.Affinity.PodAntiAffinity != nil {
		for _, elem := range poolParams.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
			podAntiAffinityTerms = append(podAntiAffinityTerms, parseModelPodAffinityTerm(elem))
		}
		for _, elem := range poolParams.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			wAffinityTerm := corev1.WeightedPodAffinityTerm{
				Weight:          *elem.Weight,
				PodAffinityTerm: parseModelPodAffinityTerm(elem.PodAffinityTerm),
			}
			weightedPodAntiAffinityTerms = append(weightedPodAntiAffinityTerms, wAffinityTerm)
		}
	}
	var podAntiAffinity *corev1.PodAntiAffinity
	if len(podAntiAffinityTerms) > 0 || len(weightedPodAntiAffinityTerms) > 0 {
		podAntiAffinity = &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution:  podAntiAffinityTerms,
			PreferredDuringSchedulingIgnoredDuringExecution: weightedPodAntiAffinityTerms,
		}
	}

	var affinity *corev1.Affinity
	if nodeAffinity != nil || podAffinity != nil || podAntiAffinity != nil {
		affinity = &corev1.Affinity{
			NodeAffinity:    nodeAffinity,
			PodAffinity:     podAffinity,
			PodAntiAffinity: podAntiAffinity,
		}
	}

	// parse tolerations
	tolerations := []corev1.Toleration{}
	for _, elem := range poolParams.Tolerations {
		var tolerationSeconds *int64
		if elem.TolerationSeconds != nil {
			// elem.TolerationSeconds.Seconds is allowed to be nil
			tolerationSeconds = elem.TolerationSeconds.Seconds
		}

		toleration := corev1.Toleration{
			Key:               elem.Key,
			Operator:          corev1.TolerationOperator(elem.Operator),
			Value:             elem.Value,
			Effect:            corev1.TaintEffect(elem.Effect),
			TolerationSeconds: tolerationSeconds,
		}
		tolerations = append(tolerations, toleration)
	}

	// Pass annotations to the volume
	vct := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "data",
			Labels:      poolParams.VolumeConfiguration.Labels,
			Annotations: poolParams.VolumeConfiguration.Annotations,
		},
		Spec: volTemp,
	}

	pool := &miniov2.Pool{
		Name:                poolParams.Name,
		Servers:             int32(*poolParams.Servers),
		VolumesPerServer:    *poolParams.VolumesPerServer,
		VolumeClaimTemplate: vct,
		Resources: corev1.ResourceRequirements{
			Requests: resourcesRequests,
			Limits:   resourcesLimits,
		},
		NodeSelector: poolParams.NodeSelector,
		Affinity:     affinity,
		Tolerations:  tolerations,
	}
	// if security context for Tenant is present, configure it.
	if poolParams.SecurityContext != nil {
		sc, err := convertModelSCToK8sSC(poolParams.SecurityContext)
		if err != nil {
			return nil, err
		}
		pool.SecurityContext = sc
	}
	return pool, nil
}

func parseModelPodAffinityTerm(term *models.PodAffinityTerm) corev1.PodAffinityTerm {
	labelMatchExpressions := []metav1.LabelSelectorRequirement{}
	for _, exp := range term.LabelSelector.MatchExpressions {
		labelSelectorReq := metav1.LabelSelectorRequirement{
			Key:      *exp.Key,
			Operator: metav1.LabelSelectorOperator(*exp.Operator),
			Values:   exp.Values,
		}
		labelMatchExpressions = append(labelMatchExpressions, labelSelectorReq)
	}

	podAffinityTerm := corev1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{
			MatchExpressions: labelMatchExpressions,
			MatchLabels:      term.LabelSelector.MatchLabels,
		},
		Namespaces:  term.Namespaces,
		TopologyKey: *term.TopologyKey,
	}
	return podAffinityTerm
}

func parseModelsNodeSelectorTerm(elem *models.NodeSelectorTerm) corev1.NodeSelectorTerm {
	var term corev1.NodeSelectorTerm
	for _, matchExpression := range elem.MatchExpressions {
		matchExp := corev1.NodeSelectorRequirement{
			Key:      *matchExpression.Key,
			Operator: corev1.NodeSelectorOperator(*matchExpression.Operator),
			Values:   matchExpression.Values,
		}
		term.MatchExpressions = append(term.MatchExpressions, matchExp)
	}
	for _, matchField := range elem.MatchFields {
		matchF := corev1.NodeSelectorRequirement{
			Key:      *matchField.Key,
			Operator: corev1.NodeSelectorOperator(*matchField.Operator),
			Values:   matchField.Values,
		}
		term.MatchFields = append(term.MatchFields, matchF)
	}
	return term
}

// parseTenantPool miniov2 pool object and returns the equivalent
// models.Pool object
func parseTenantPool(pool *miniov2.Pool) *models.Pool {
	var size *int64
	var storageClassName string
	if pool.VolumeClaimTemplate != nil {
		size = swag.Int64(pool.VolumeClaimTemplate.Spec.Resources.Requests.Storage().Value())
		if pool.VolumeClaimTemplate.Spec.StorageClassName != nil {
			storageClassName = *pool.VolumeClaimTemplate.Spec.StorageClassName
		}
	}

	// parse resources' requests
	var resources *models.PoolResources
	resourcesRequests := make(map[string]int64)
	resourcesLimits := make(map[string]int64)
	for key, val := range pool.Resources.Requests {
		resourcesRequests[key.String()] = val.Value()
	}
	for key, val := range pool.Resources.Limits {
		resourcesLimits[key.String()] = val.Value()
	}
	if len(resourcesRequests) > 0 || len(resourcesLimits) > 0 {
		resources = &models.PoolResources{
			Limits:   resourcesLimits,
			Requests: resourcesRequests,
		}
	}

	// parse Node Affinity
	nodeSelectorTerms := []*models.NodeSelectorTerm{}
	preferredSchedulingTerm := []*models.PoolAffinityNodeAffinityPreferredDuringSchedulingIgnoredDuringExecutionItems0{}

	if pool.Affinity != nil && pool.Affinity.NodeAffinity != nil {
		if pool.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			for _, elem := range pool.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
				term := parseNodeSelectorTerm(&elem)
				nodeSelectorTerms = append(nodeSelectorTerms, term)
			}
		}
		for _, elem := range pool.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			pst := &models.PoolAffinityNodeAffinityPreferredDuringSchedulingIgnoredDuringExecutionItems0{
				Weight:     swag.Int32(elem.Weight),
				Preference: parseNodeSelectorTerm(&elem.Preference),
			}
			preferredSchedulingTerm = append(preferredSchedulingTerm, pst)
		}
	}

	var nodeAffinity *models.PoolAffinityNodeAffinity
	if len(nodeSelectorTerms) > 0 || len(preferredSchedulingTerm) > 0 {
		nodeAffinity = &models.PoolAffinityNodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &models.PoolAffinityNodeAffinityRequiredDuringSchedulingIgnoredDuringExecution{
				NodeSelectorTerms: nodeSelectorTerms,
			},
			PreferredDuringSchedulingIgnoredDuringExecution: preferredSchedulingTerm,
		}
	}

	// parse Pod Affinity
	podAffinityTerms := []*models.PodAffinityTerm{}
	weightedPodAffinityTerms := []*models.PoolAffinityPodAffinityPreferredDuringSchedulingIgnoredDuringExecutionItems0{}

	if pool.Affinity != nil && pool.Affinity.PodAffinity != nil {
		for _, elem := range pool.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
			podAffinityTerms = append(podAffinityTerms, parsePodAffinityTerm(&elem))
		}
		for _, elem := range pool.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			wAffinityTerm := &models.PoolAffinityPodAffinityPreferredDuringSchedulingIgnoredDuringExecutionItems0{
				Weight:          swag.Int32(elem.Weight),
				PodAffinityTerm: parsePodAffinityTerm(&elem.PodAffinityTerm),
			}
			weightedPodAffinityTerms = append(weightedPodAffinityTerms, wAffinityTerm)
		}
	}
	var podAffinity *models.PoolAffinityPodAffinity
	if len(podAffinityTerms) > 0 || len(weightedPodAffinityTerms) > 0 {
		podAffinity = &models.PoolAffinityPodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution:  podAffinityTerms,
			PreferredDuringSchedulingIgnoredDuringExecution: weightedPodAffinityTerms,
		}
	}

	// parse Pod Anti Affinity
	podAntiAffinityTerms := []*models.PodAffinityTerm{}
	weightedPodAntiAffinityTerms := []*models.PoolAffinityPodAntiAffinityPreferredDuringSchedulingIgnoredDuringExecutionItems0{}

	if pool.Affinity != nil && pool.Affinity.PodAntiAffinity != nil {
		for _, elem := range pool.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
			podAntiAffinityTerms = append(podAntiAffinityTerms, parsePodAffinityTerm(&elem))
		}
		for _, elem := range pool.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			wAffinityTerm := &models.PoolAffinityPodAntiAffinityPreferredDuringSchedulingIgnoredDuringExecutionItems0{
				Weight:          swag.Int32(elem.Weight),
				PodAffinityTerm: parsePodAffinityTerm(&elem.PodAffinityTerm),
			}
			weightedPodAntiAffinityTerms = append(weightedPodAntiAffinityTerms, wAffinityTerm)
		}
	}

	var podAntiAffinity *models.PoolAffinityPodAntiAffinity
	if len(podAntiAffinityTerms) > 0 || len(weightedPodAntiAffinityTerms) > 0 {
		podAntiAffinity = &models.PoolAffinityPodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution:  podAntiAffinityTerms,
			PreferredDuringSchedulingIgnoredDuringExecution: weightedPodAntiAffinityTerms,
		}
	}

	// build affinity object
	var affinity *models.PoolAffinity
	if nodeAffinity != nil || podAffinity != nil || podAntiAffinity != nil {
		affinity = &models.PoolAffinity{
			NodeAffinity:    nodeAffinity,
			PodAffinity:     podAffinity,
			PodAntiAffinity: podAntiAffinity,
		}
	}

	// parse tolerations
	var tolerations models.PoolTolerations
	for _, elem := range pool.Tolerations {
		var tolerationSecs *models.PoolTolerationSeconds
		if elem.TolerationSeconds != nil {
			tolerationSecs = &models.PoolTolerationSeconds{
				Seconds: elem.TolerationSeconds,
			}
		}
		toleration := &models.PoolTolerationsItems0{
			Key:               elem.Key,
			Operator:          string(elem.Operator),
			Value:             elem.Value,
			Effect:            string(elem.Effect),
			TolerationSeconds: tolerationSecs,
		}
		tolerations = append(tolerations, toleration)
	}

	poolModel := &models.Pool{
		Name:             pool.Name,
		Servers:          swag.Int64(int64(pool.Servers)),
		VolumesPerServer: swag.Int32(pool.VolumesPerServer),
		VolumeConfiguration: &models.PoolVolumeConfiguration{
			Size:             size,
			StorageClassName: storageClassName,
		},
		NodeSelector: pool.NodeSelector,
		Resources:    resources,
		Affinity:     affinity,
		Tolerations:  tolerations,
	}
	return poolModel
}

func parsePodAffinityTerm(term *corev1.PodAffinityTerm) *models.PodAffinityTerm {
	labelMatchExpressions := []*models.PodAffinityTermLabelSelectorMatchExpressionsItems0{}
	for _, exp := range term.LabelSelector.MatchExpressions {
		labelSelectorReq := &models.PodAffinityTermLabelSelectorMatchExpressionsItems0{
			Key:      swag.String(exp.Key),
			Operator: swag.String(string(exp.Operator)),
			Values:   exp.Values,
		}
		labelMatchExpressions = append(labelMatchExpressions, labelSelectorReq)
	}

	podAffinityTerm := &models.PodAffinityTerm{
		LabelSelector: &models.PodAffinityTermLabelSelector{
			MatchExpressions: labelMatchExpressions,
			MatchLabels:      term.LabelSelector.MatchLabels,
		},
		Namespaces:  term.Namespaces,
		TopologyKey: swag.String(term.TopologyKey),
	}
	return podAffinityTerm
}

func parseNodeSelectorTerm(term *corev1.NodeSelectorTerm) *models.NodeSelectorTerm {
	var t models.NodeSelectorTerm
	for _, matchExpression := range term.MatchExpressions {
		matchExp := &models.NodeSelectorTermMatchExpressionsItems0{
			Key:      swag.String(matchExpression.Key),
			Operator: swag.String(string(matchExpression.Operator)),
			Values:   matchExpression.Values,
		}
		t.MatchExpressions = append(t.MatchExpressions, matchExp)
	}
	for _, matchField := range term.MatchFields {
		matchF := &models.NodeSelectorTermMatchFieldsItems0{
			Key:      swag.String(matchField.Key),
			Operator: swag.String(string(matchField.Operator)),
			Values:   matchField.Values,
		}
		t.MatchFields = append(t.MatchFields, matchF)
	}
	return &t
}

func getTenantUpdatePoolResponse(session *models.Principal, params operator_api.TenantUpdatePoolsParams) (*models.Tenant, *models.Error) {
	ctx := context.Background()
	opClientClientSet, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return nil, prepareError(err)
	}

	opClient := &operatorClient{
		client: opClientClientSet,
	}

	t, err := updateTenantPools(ctx, opClient, params.Namespace, params.Tenant, params.Body.Pools)
	if err != nil {
		restapi.LogError("error updating Tenant's pools: %v", err)
		return nil, prepareError(err)
	}

	// parse it to models.Tenant
	tenant := getTenantInfo(t)
	return tenant, nil
}

// updateTenantPools Sets the Tenant's pools to the ones provided by the request
//
// It does the equivalent to a PUT request on Tenant's pools
func updateTenantPools(
	ctx context.Context,
	operatorClient OperatorClientI,
	namespace string,
	tenantName string,
	poolsReq []*models.Pool) (*miniov2.Tenant, error) {

	minInst, err := operatorClient.TenantGet(ctx, namespace, tenantName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// set the pools if they are provided
	var newPoolArray []miniov2.Pool
	for _, pool := range poolsReq {
		pool, err := parseTenantPoolRequest(pool)
		if err != nil {
			return nil, err
		}
		newPoolArray = append(newPoolArray, *pool)
	}

	// replace pools array
	minInst.Spec.Pools = newPoolArray

	minInst = minInst.DeepCopy()
	minInst.EnsureDefaults()

	payloadBytes, err := json.Marshal(minInst)
	if err != nil {
		return nil, err
	}
	tenantUpdated, err := operatorClient.TenantPatch(ctx, namespace, minInst.Name, types.MergePatchType, payloadBytes, metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}
	return tenantUpdated, nil
}

func getTenantYAML(session *models.Principal, params operator_api.GetTenantYAMLParams) (*models.TenantYAML, *models.Error) {
	// get Kubernetes Client

	opClient, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return nil, prepareError(err)
	}

	tenant, err := opClient.MinioV2().Tenants(params.Namespace).Get(params.HTTPRequest.Context(), params.Tenant, metav1.GetOptions{})
	if err != nil {
		return nil, prepareError(err)
	}
	// remove managed fields
	tenant.ManagedFields = []metav1.ManagedFieldsEntry{}

	//yb, err := yaml.Marshal(tenant)
	serializer := k8sJson.NewSerializerWithOptions(
		k8sJson.DefaultMetaFactory, nil, nil,
		k8sJson.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		},
	)
	buf := new(bytes.Buffer)

	err = serializer.Encode(tenant, buf)
	if err != nil {
		return nil, prepareError(err)
	}

	yb := buf.String()

	return &models.TenantYAML{Yaml: yb}, nil
}

func getUpdateTenantYAML(session *models.Principal, params operator_api.PutTenantYAMLParams) *models.Error {
	// https://godoc.org/k8s.io/apimachinery/pkg/runtime#Scheme
	scheme := runtime.NewScheme()

	// https://godoc.org/k8s.io/apimachinery/pkg/runtime/serializer#CodecFactory
	codecFactory := serializer.NewCodecFactory(scheme)

	// https://godoc.org/k8s.io/apimachinery/pkg/runtime#Decoder
	deserializer := codecFactory.UniversalDeserializer()

	tenantObject, _, err := deserializer.Decode([]byte(params.Body.Yaml), nil, &miniov2.Tenant{})
	if err != nil {
		return &models.Error{Code: 400, Message: swag.String(err.Error())}
	}
	inTenant := tenantObject.(*miniov2.Tenant)
	// get Kubernetes Client
	opClient, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return prepareError(err)
	}

	tenant, err := opClient.MinioV2().Tenants(params.Namespace).Get(params.HTTPRequest.Context(), params.Tenant, metav1.GetOptions{})
	if err != nil {
		return prepareError(err)
	}
	upTenant := tenant.DeepCopy()
	// only update safe fields: spec, metadata.finalizers, metadata.labels and metadata.annotations
	upTenant.Labels = inTenant.Labels
	upTenant.Annotations = inTenant.Annotations
	upTenant.Finalizers = inTenant.Finalizers
	upTenant.Spec = inTenant.Spec

	_, err = opClient.MinioV2().Tenants(params.Namespace).Update(params.HTTPRequest.Context(), upTenant, metav1.UpdateOptions{})
	if err != nil {
		return &models.Error{Code: 400, Message: swag.String(err.Error())}
	}

	return nil
}

func getTenantEventsResponse(session *models.Principal, params operator_api.GetTenantEventsParams) (models.EventListWrapper, *models.Error) {
	ctx := context.Background()
	client, err := cluster.OperatorClient(session.STSSessionToken)
	if err != nil {
		return nil, prepareError(err)
	}
	clientset, err := cluster.K8sClient(session.STSSessionToken)
	if err != nil {
		return nil, prepareError(err)
	}
	tenant, err := client.MinioV2().Tenants(params.Namespace).Get(ctx, params.Tenant, metav1.GetOptions{})
	if err != nil {
		return nil, prepareError(err)
	}
	events, err := clientset.CoreV1().Events(params.Namespace).List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.uid=%s", tenant.UID)})
	if err != nil {
		return nil, prepareError(err)
	}
	retval := models.EventListWrapper{}
	for _, event := range events.Items {
		retval = append(retval, &models.EventListElement{
			Namespace: event.Namespace,
			LastSeen:  event.LastTimestamp.Unix(),
			Message:   event.Message,
			EventType: event.Type,
			Reason:    event.Reason,
		})
	}
	sort.SliceStable(retval, func(i int, j int) bool {
		return retval[i].LastSeen < retval[j].LastSeen
	})
	return retval, nil
}
