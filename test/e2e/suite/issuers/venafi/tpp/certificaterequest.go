/*
Copyright 2020 The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tpp

import (
	"context"
	"crypto/x509"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/cert-manager/cert-manager/e2e-tests/framework"
	vaddon "github.com/cert-manager/cert-manager/e2e-tests/framework/addon/venafi"
	"github.com/cert-manager/cert-manager/e2e-tests/util"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/cert-manager/cert-manager/test/unit/gen"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = TPPDescribe("CertificateRequest with a properly configured Issuer", func() {
	f := framework.NewDefaultFramework("venafi-tpp-certificaterequest")
	h := f.Helper()

	var (
		issuer                 *cmapi.Issuer
		tppAddon               = &vaddon.VenafiTPP{}
		certificateRequestName = "test-venafi-certificaterequest"
	)

	BeforeEach(func(testingCtx context.Context) {
		tppAddon.Namespace = f.Namespace.Name
	})

	f.RequireAddon(tppAddon)

	// Create the Issuer resource
	BeforeEach(func(testingCtx context.Context) {
		var err error

		By("Creating a Venafi Issuer resource")
		issuer = tppAddon.Details().BuildIssuer()
		issuer, err = f.CertManagerClientSet.CertmanagerV1().Issuers(f.Namespace.Name).Create(testingCtx, issuer, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for Issuer to become Ready")
		err = util.WaitForIssuerCondition(testingCtx, f.CertManagerClientSet.CertmanagerV1().Issuers(f.Namespace.Name),
			issuer.Name,
			cmapi.IssuerCondition{
				Type:   cmapi.IssuerConditionReady,
				Status: cmmeta.ConditionTrue,
			})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func(testingCtx context.Context) {
		By("Cleaning up")
		if issuer != nil {
			err := f.CertManagerClientSet.CertmanagerV1().Issuers(f.Namespace.Name).Delete(testingCtx, issuer.Name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("should obtain a signed certificate for a single domain", func(testingCtx context.Context) {
		crClient := f.CertManagerClientSet.CertmanagerV1().CertificateRequests(f.Namespace.Name)

		dnsNames := []string{rand.String(10) + ".venafi-e2e.example"}

		csr, key, err := gen.CSR(x509.RSA, gen.SetCSRCommonName(dnsNames[0]), gen.SetCSRDNSNames(dnsNames...))
		Expect(err).NotTo(HaveOccurred())
		cr := gen.CertificateRequest(certificateRequestName,
			gen.SetCertificateRequestNamespace(f.Namespace.Name),
			gen.SetCertificateRequestIssuer(cmmeta.ObjectReference{Kind: cmapi.IssuerKind, Name: issuer.Name}),
			gen.SetCertificateRequestCSR(csr),
		)

		By("Creating a CertificateRequest")
		_, err = crClient.Create(testingCtx, cr, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Verifying the CertificateRequest is valid")
		err = h.WaitCertificateRequestIssuedValid(testingCtx, f.Namespace.Name, certificateRequestName, time.Second*30, key)
		Expect(err).NotTo(HaveOccurred())
	})
})
