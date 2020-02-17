package resource_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	rabbitmqv1beta1 "github.com/pivotal/rabbitmq-for-kubernetes/api/v1beta1"
	"github.com/pivotal/rabbitmq-for-kubernetes/internal/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	defaultscheme "k8s.io/client-go/kubernetes/scheme"
)

var _ = Context("IngressServices", func() {
	var (
		instance   rabbitmqv1beta1.RabbitmqCluster
		rmqBuilder resource.RabbitmqResourceBuilder
		scheme     *runtime.Scheme
	)

	Context("Build", func() {
		BeforeEach(func() {
			scheme = runtime.NewScheme()
			Expect(rabbitmqv1beta1.AddToScheme(scheme)).To(Succeed())
			Expect(defaultscheme.AddToScheme(scheme)).To(Succeed())
			instance = generateRabbitmqCluster()
			rmqBuilder = resource.RabbitmqResourceBuilder{
				Instance: &instance,
				Scheme:   scheme,
			}
		})

		It("Builds using the values from the CR", func() {
			serviceBuilder := rmqBuilder.IngressService()
			obj, err := serviceBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			service := obj.(*corev1.Service)

			By("generates a service object with the correct name and labels", func() {
				expectedName := instance.ChildResourceName("ingress")
				Expect(service.Name).To(Equal(expectedName))
			})

			By("generates a service object with the correct namespace", func() {
				Expect(service.Namespace).To(Equal(instance.Namespace))
			})
		})
	})

	Context("Update", func() {
		BeforeEach(func() {
			scheme = runtime.NewScheme()
			Expect(rabbitmqv1beta1.AddToScheme(scheme)).To(Succeed())
			Expect(defaultscheme.AddToScheme(scheme)).To(Succeed())
			instance = generateRabbitmqCluster()
			rmqBuilder = resource.RabbitmqResourceBuilder{
				Instance: &instance,
				Scheme:   scheme,
			}
		})

		Context("Annotations", func() {
			When("CR instance does have service annotations specified", func() {
				It("generates a service object with the annotations as specified", func() {
					serviceAnno := map[string]string{
						"service_annotation_a":       "0.0.0.0/0",
						"kubernetes.io/name":         "i-do-not-like-this",
						"kubectl.kubernetes.io/name": "i-do-not-like-this",
						"k8s.io/name":                "i-do-not-like-this",
					}
					expectedAnnotations := map[string]string{
						"service_annotation_a":             "0.0.0.0/0",
						"app.kubernetes.io/part-of":        "pivotal-rabbitmq",
						"app.k8s.io/something":             "something-amazing",
						"this-was-the-previous-annotation": "should-be-preserved",
					}

					service := updateServiceWithAnnotations(rmqBuilder, nil, serviceAnno)
					Expect(service.ObjectMeta.Annotations).To(Equal(expectedAnnotations))
				})
			})

			When("CR instance does not have service annotations specified", func() {
				It("generates the service annotations as specified", func() {
					expectedAnnotations := map[string]string{
						"app.kubernetes.io/part-of":        "pivotal-rabbitmq",
						"app.k8s.io/something":             "something-amazing",
						"this-was-the-previous-annotation": "should-be-preserved",
					}

					var serviceAnnotations map[string]string = nil
					var instanceAnnotations map[string]string = nil
					service := updateServiceWithAnnotations(rmqBuilder, instanceAnnotations, serviceAnnotations)
					Expect(service.ObjectMeta.Annotations).To(Equal(expectedAnnotations))
				})
			})

			When("CR instance does not have service annotations specified, but does have metadata annotations specified", func() {
				It("sets the instance annotations on the service", func() {
					instanceMetadataAnnotations := map[string]string{
						"my-annotation":              "i-like-this",
						"kubernetes.io/name":         "i-do-not-like-this",
						"kubectl.kubernetes.io/name": "i-do-not-like-this",
						"k8s.io/name":                "i-do-not-like-this",
					}

					var serviceAnnotations map[string]string = nil
					service := updateServiceWithAnnotations(rmqBuilder, instanceMetadataAnnotations, serviceAnnotations)
					expectedAnnotations := map[string]string{
						"my-annotation":                    "i-like-this",
						"app.kubernetes.io/part-of":        "pivotal-rabbitmq",
						"app.k8s.io/something":             "something-amazing",
						"this-was-the-previous-annotation": "should-be-preserved",
					}

					Expect(service.Annotations).To(Equal(expectedAnnotations))
				})
			})

			When("CR instance has service annotations specified, and has metadata annotations specified", func() {
				It("merges the annotations", func() {
					serviceAnnotations := map[string]string{
						"service_annotation_a":       "0.0.0.0/0",
						"my-annotation":              "i-like-this-more",
						"kubernetes.io/name":         "i-do-not-like-this",
						"kubectl.kubernetes.io/name": "i-do-not-like-this",
						"k8s.io/name":                "i-do-not-like-this",
					}
					instanceAnnotations := map[string]string{
						"my-annotation":              "i-like-this",
						"my-second-annotation":       "i-like-this-also",
						"kubernetes.io/name":         "i-do-not-like-this",
						"kubectl.kubernetes.io/name": "i-do-not-like-this",
						"k8s.io/name":                "i-do-not-like-this",
					}

					expectedAnnotations := map[string]string{
						"my-annotation":                    "i-like-this-more",
						"my-second-annotation":             "i-like-this-also",
						"service_annotation_a":             "0.0.0.0/0",
						"app.kubernetes.io/part-of":        "pivotal-rabbitmq",
						"app.k8s.io/something":             "something-amazing",
						"this-was-the-previous-annotation": "should-be-preserved",
					}

					service := updateServiceWithAnnotations(rmqBuilder, instanceAnnotations, serviceAnnotations)

					Expect(service.ObjectMeta.Annotations).To(Equal(expectedAnnotations))
				})
			})
		})

		Context("Labels", func() {
			var (
				serviceBuilder *resource.IngressServiceBuilder
				ingressService *corev1.Service
			)
			BeforeEach(func() {
				serviceBuilder = rmqBuilder.IngressService()
				instance = rabbitmqv1beta1.RabbitmqCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "rabbit-labelled",
					},
				}
				instance.Labels = map[string]string{
					"app.kubernetes.io/foo": "bar",
					"foo":                   "bar",
					"rabbitmq":              "is-great",
					"foo/app.kubernetes.io": "edgecase",
				}

				ingressService = &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.kubernetes.io/name":      instance.Name,
							"app.kubernetes.io/part-of":   "pivotal-rabbitmq",
							"this-was-the-previous-label": "should-be-deleted",
						},
					},
				}
				err := serviceBuilder.Update(ingressService)
				Expect(err).NotTo(HaveOccurred())
			})

			It("adds labels from the CR", func() {
				testLabels(ingressService.Labels)
			})

			It("restores the default labels", func() {
				labels := ingressService.Labels
				Expect(labels["app.kubernetes.io/name"]).To(Equal(instance.Name))
				Expect(labels["app.kubernetes.io/component"]).To(Equal("rabbitmq"))
				Expect(labels["app.kubernetes.io/part-of"]).To(Equal("pivotal-rabbitmq"))
			})

			It("deletes the labels that are removed from the CR", func() {
				Expect(ingressService.Labels).NotTo(HaveKey("this-was-the-previous-label"))
			})
		})

		Context("Service Type", func() {
			var (
				ingressService *corev1.Service
				serviceBuilder *resource.IngressServiceBuilder
			)

			BeforeEach(func() {
				serviceBuilder = rmqBuilder.IngressService()
				instance = generateRabbitmqCluster()

				ingressService = &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rabbit-service-type-update-service",
						Namespace: "foo-namespace",
					},
				}
			})

			It("sets the service type to the value specified in the CR instance by default", func() {
				err := serviceBuilder.Update(ingressService)
				Expect(err).NotTo(HaveOccurred())

				expectedServiceType := "this-is-a-service"
				Expect(string(ingressService.Spec.Type)).To(Equal(expectedServiceType))
			})

			It("adds a label selector with the instance name", func() {
				err := serviceBuilder.Update(ingressService)
				Expect(err).NotTo(HaveOccurred())

				Expect(ingressService.Spec.Selector["app.kubernetes.io/name"]).To(Equal(instance.Name))
			})

			It("sets the onwer reference", func() {
				err := serviceBuilder.Update(ingressService)
				Expect(err).NotTo(HaveOccurred())

				Expect(ingressService.ObjectMeta.OwnerReferences[0].Name).To(Equal("foo"))
			})

			It("exposes the required ports", func() {
				err := serviceBuilder.Update(ingressService)
				Expect(err).NotTo(HaveOccurred())

				amqpPort := corev1.ServicePort{
					Name:     "amqp",
					Port:     5672,
					Protocol: corev1.ProtocolTCP,
				}
				httpPort := corev1.ServicePort{
					Name:     "http",
					Port:     15672,
					Protocol: corev1.ProtocolTCP,
				}
				prometheusPort := corev1.ServicePort{
					Name:     "prometheus",
					Port:     15692,
					Protocol: corev1.ProtocolTCP,
				}
				Expect(ingressService.Spec.Ports).Should(ConsistOf(amqpPort, httpPort, prometheusPort))
			})

			It("updates the service type from ClusterIP to NodePort", func() {
				ingressService.Spec.Type = corev1.ServiceTypeClusterIP
				serviceBuilder.Instance.Spec.Service.Type = "NodePort"
				err := serviceBuilder.Update(ingressService)
				Expect(err).NotTo(HaveOccurred())

				expectedServiceType := "NodePort"
				Expect(string(ingressService.Spec.Type)).To(Equal(expectedServiceType))
			})

			It("preserves the same node ports after updating from LoadBalancer to NodePort", func() {
				ingressService.Spec.Type = corev1.ServiceTypeLoadBalancer
				ingressService.Spec.Ports = []corev1.ServicePort{
					corev1.ServicePort{
						Protocol: corev1.ProtocolTCP,
						Port:     5672,
						Name:     "amqp",
						NodePort: 12345,
					},
					corev1.ServicePort{
						Protocol: corev1.ProtocolTCP,
						Port:     15672,
						Name:     "http",
						NodePort: 1234,
					},
					corev1.ServicePort{
						Protocol: corev1.ProtocolTCP,
						Port:     15692,
						Name:     "prometheus",
						NodePort: 2345,
					},
				}

				serviceBuilder.Instance.Spec.Service.Type = "NodePort"
				err := serviceBuilder.Update(ingressService)
				Expect(err).NotTo(HaveOccurred())

				expectedAmqpServicePort := corev1.ServicePort{
					Name:     "amqp",
					Protocol: corev1.ProtocolTCP,
					Port:     5672,
					NodePort: 12345,
				}
				expectedHttpServicePort := corev1.ServicePort{
					Protocol: corev1.ProtocolTCP,
					Port:     15672,
					Name:     "http",
					NodePort: 1234,
				}
				expectedPrometheusServicePort := corev1.ServicePort{
					Protocol: corev1.ProtocolTCP,
					Port:     15692,
					Name:     "prometheus",
					NodePort: 2345,
				}

				Expect(ingressService.Spec.Ports).To(ContainElement(expectedAmqpServicePort))
				Expect(ingressService.Spec.Ports).To(ContainElement(expectedHttpServicePort))
				Expect(ingressService.Spec.Ports).To(ContainElement(expectedPrometheusServicePort))
			})

			It("resets ports", func() {})

			It("unsets nodePort after updating from NodePort to ClusterIP", func() {
				ingressService.Spec.Type = corev1.ServiceTypeNodePort
				ingressService.Spec.Ports = []corev1.ServicePort{
					corev1.ServicePort{
						Protocol: corev1.ProtocolTCP,
						Port:     5672,
						Name:     "amqp",
						NodePort: 12345,
					},
				}

				serviceBuilder.Instance.Spec.Service.Type = "ClusterIP"
				err := serviceBuilder.Update(ingressService)
				Expect(err).NotTo(HaveOccurred())

				// We cant set nodePort to nil because its a primitive
				// For Kubernetes API, setting it to 0 is the same as not setting it at all
				expectedServicePort := corev1.ServicePort{
					Name:     "amqp",
					Protocol: corev1.ProtocolTCP,
					Port:     5672,
					NodePort: 0,
				}

				Expect(ingressService.Spec.Ports).To(ContainElement(expectedServicePort))
			})

			It("unsets the service type and node ports when service type is deleted from CR spec", func() {
				ingressService.Spec.Type = corev1.ServiceTypeNodePort
				ingressService.Spec.Ports = []corev1.ServicePort{
					corev1.ServicePort{
						Protocol: corev1.ProtocolTCP,
						Port:     5672,
						Name:     "amqp",
						NodePort: 12345,
					},
				}

				serviceBuilder.Instance.Spec.Service.Type = ""
				err := serviceBuilder.Update(ingressService)
				Expect(err).NotTo(HaveOccurred())

				expectedServicePort := corev1.ServicePort{
					Name:     "amqp",
					Protocol: corev1.ProtocolTCP,
					Port:     5672,
					NodePort: 0,
				}

				Expect(ingressService.Spec.Ports).To(ContainElement(expectedServicePort))
			})
		})
	})
})

func updateServiceWithAnnotations(rmqBuilder resource.RabbitmqResourceBuilder, instanceAnnotations, serviceAnnotations map[string]string) *corev1.Service {
	instance := &rabbitmqv1beta1.RabbitmqCluster{
		ObjectMeta: v1.ObjectMeta{
			Name:        "foo",
			Namespace:   "foo-namespace",
			Annotations: instanceAnnotations,
		},
		Spec: rabbitmqv1beta1.RabbitmqClusterSpec{
			Service: rabbitmqv1beta1.RabbitmqClusterServiceSpec{
				Annotations: serviceAnnotations,
			},
		},
	}

	rmqBuilder.Instance = instance
	serviceBuilder := rmqBuilder.IngressService()
	ingressService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-service",
			Namespace: "foo-namespace",
			Annotations: map[string]string{
				"this-was-the-previous-annotation": "should-be-preserved",
				"app.kubernetes.io/part-of":        "pivotal-rabbitmq",
				"app.k8s.io/something":             "something-amazing",
			},
		},
	}
	Expect(serviceBuilder.Update(ingressService)).To(Succeed())
	return ingressService
}
