package remote_test

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kong/kuma/api/mesh/v1alpha1"
	"github.com/Kong/kuma/pkg/core/resources/apis/mesh"
	errors_types "github.com/Kong/kuma/pkg/core/rest/errors/types"
	sample_api "github.com/Kong/kuma/pkg/test/apis/sample/v1alpha1"
	"github.com/Kong/kuma/pkg/test/resources/model"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	core_model "github.com/Kong/kuma/pkg/core/resources/model"
	core_rest "github.com/Kong/kuma/pkg/core/resources/model/rest"
	core_store "github.com/Kong/kuma/pkg/core/resources/store"
	"github.com/Kong/kuma/pkg/plugins/resources/remote"

	sample_core "github.com/Kong/kuma/pkg/test/resources/apis/sample"
)

var _ = Describe("RemoteStore", func() {

	type RequestAssertion = func(req *http.Request)

	setupStore := func(file string, assertion RequestAssertion) core_store.ResourceStore {
		client := &http.Client{
			Transport: RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assertion(req)

				file, err := os.Open(filepath.Join("testdata", file))
				if err != nil {
					return nil, err
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bufio.NewReader(file)),
				}, nil
			}),
		}
		apis := &core_rest.ApiDescriptor{
			Resources: map[core_model.ResourceType]core_rest.ResourceApi{
				sample_core.TrafficRouteType: core_rest.NewResourceApi(sample_core.TrafficRouteType, "traffic-routes"),
				mesh.MeshType:                core_rest.NewResourceApi(mesh.MeshType, "meshes"),
			},
		}
		return remote.NewStore(client, apis)
	}

	setupErrorStore := func(code int, errorMsg string) core_store.ResourceStore {
		client := &http.Client{
			Transport: RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: code,
					Body:       ioutil.NopCloser(strings.NewReader(errorMsg)),
				}, nil
			}),
		}
		apis := &core_rest.ApiDescriptor{
			Resources: map[core_model.ResourceType]core_rest.ResourceApi{
				sample_core.TrafficRouteType: core_rest.NewResourceApi(sample_core.TrafficRouteType, "traffic-routes"),
				mesh.MeshType:                core_rest.NewResourceApi(mesh.MeshType, "meshes"),
			},
		}
		return remote.NewStore(client, apis)
	}

	Describe("Get()", func() {
		It("should get resource", func() {
			// setup
			name := "res-1"
			store := setupStore("get.json", func(req *http.Request) {
				Expect(req.URL.Path).To(Equal(fmt.Sprintf("/meshes/default/traffic-routes/%s", name)))
			})

			// when
			resource := sample_core.TrafficRouteResource{}
			err := store.Get(context.Background(), &resource, core_store.GetByKey(name, "default"))

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(resource.Spec.Path).To(Equal("/example"))

			Expect(resource.GetMeta().GetName()).To(Equal("res-1"))
			Expect(resource.GetMeta().GetMesh()).To(Equal("default"))
		})

		It("should get mesh resource", func() {
			meshName := "someMesh"
			store := setupStore("get-mesh.json", func(req *http.Request) {
				Expect(req.URL.Path).To(Equal(fmt.Sprintf("/meshes/%s", meshName)))
			})

			// when
			resource := mesh.MeshResource{}
			err := store.Get(context.Background(), &resource, core_store.GetByKey(meshName, meshName))

			// then
			Expect(err).ToNot(HaveOccurred())

			Expect(resource.GetMeta().GetName()).To(Equal(meshName))
			Expect(resource.GetMeta().GetMesh()).To(Equal(meshName))
		})

		It("should parse kuma api server error", func() {
			json := `
			{
				"title": "Could not get resource",
				"details": "Internal Server Error"
			}
		`
			store := setupErrorStore(400, json)

			// when
			resource := mesh.MeshResource{}
			err := store.Get(context.Background(), &resource, core_store.GetByKey("test", "test"))

			// then
			Expect(err).To(HaveOccurred())

			Expect(err).To(Equal(&errors_types.Error{
				Title:   "Could not get resource",
				Details: "Internal Server Error",
			}))
		})

		It("should map 404 error to ResourceNotFound", func() {
			// given
			json := `
			{
				"title": "Could not get a resource",
				"details": "Not found"
			}`
			store := setupErrorStore(404, json)

			// when
			resource := mesh.MeshResource{}
			err := store.Get(context.Background(), &resource, core_store.GetByKey("test", "test"))

			// then
			Expect(core_store.IsResourceNotFound(err)).To(BeTrue())
		})
	})

	Describe("Create()", func() {
		It("should send proper json", func() {
			// setup
			name := "res-1"
			store := setupStore("create_update.json", func(req *http.Request) {
				Expect(req.URL.Path).To(Equal(fmt.Sprintf("/meshes/default/traffic-routes/%s", name)))
				bytes, err := ioutil.ReadAll(req.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(bytes).To(MatchJSON(`{"mesh":"default","name":"res-1","path":"/some-path","type":"SampleTrafficRoute"}`))
			})

			// when
			resource := sample_core.TrafficRouteResource{
				Spec: sample_api.TrafficRoute{
					Path: "/some-path",
				},
			}
			err := store.Create(context.Background(), &resource, core_store.CreateByKey(name, "default"))

			// then
			Expect(err).ToNot(HaveOccurred())
		})

		It("should send proper mesh json", func() {
			// setup
			meshName := "someMesh"
			store := setupStore("create_update.json", func(req *http.Request) {
				Expect(req.URL.Path).To(Equal(fmt.Sprintf("/meshes/%s", meshName)))
				bytes, err := ioutil.ReadAll(req.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(bytes).To(MatchJSON(`{"mesh":"someMesh","name":"someMesh","type":"Mesh"}`))
			})

			// when
			resource := mesh.MeshResource{
				Spec: v1alpha1.Mesh{},
			}
			err := store.Create(context.Background(), &resource, core_store.CreateByKey(meshName, meshName))

			// then
			Expect(err).ToNot(HaveOccurred())
		})

		It("should parse kuma api server error", func() {
			json := `
			{
				"title": "Could not process resource",
				"details": "Resource is not valid",
				"causes": [
					{
						"field": "mtls",
						"message": "cannot be empty"
					}
				]
			}
		`
			store := setupErrorStore(400, json)

			// when
			err := store.Create(context.Background(), &mesh.MeshResource{}, core_store.CreateByKey("test", "test"))

			// then
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(&errors_types.Error{
				Title:   "Could not process resource",
				Details: "Resource is not valid",
				Causes: []errors_types.Cause{
					{
						Field:   "mtls",
						Message: "cannot be empty",
					},
				},
			}))
		})
	})

	Describe("Update()", func() {
		It("should send proper json", func() {
			// setup
			name := "res-1"
			store := setupStore("create_update.json", func(req *http.Request) {
				Expect(req.URL.Path).To(Equal(fmt.Sprintf("/meshes/default/traffic-routes/%s", name)))
				bytes, err := ioutil.ReadAll(req.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(bytes).To(MatchJSON(`{"mesh":"default","name":"res-1","path":"/some-path","type":"SampleTrafficRoute"}`))
			})

			// when
			resource := sample_core.TrafficRouteResource{
				Spec: sample_api.TrafficRoute{
					Path: "/some-path",
				},
				Meta: &model.ResourceMeta{
					Mesh: "default",
					Name: name,
				},
			}
			err := store.Update(context.Background(), &resource)

			// then
			Expect(err).ToNot(HaveOccurred())
		})

		It("should send proper mesh json", func() {
			// setup
			meshName := "someMesh"
			store := setupStore("create_update.json", func(req *http.Request) {
				Expect(req.URL.Path).To(Equal(fmt.Sprintf("/meshes/%s", meshName)))
				bytes, err := ioutil.ReadAll(req.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(bytes).To(MatchJSON(`{"mesh":"someMesh","mtls":{"ca":{"builtin":{}}},"name":"someMesh","type":"Mesh"}`))
			})

			// when
			resource := mesh.MeshResource{
				Spec: v1alpha1.Mesh{
					Mtls: &v1alpha1.Mesh_Mtls{
						Ca: &v1alpha1.CertificateAuthority{
							Type: &v1alpha1.CertificateAuthority_Builtin_{
								Builtin: &v1alpha1.CertificateAuthority_Builtin{},
							},
						},
					},
				},
				Meta: &model.ResourceMeta{
					Mesh: meshName,
					Name: meshName,
				},
			}
			err := store.Update(context.Background(), &resource)

			// then
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return error from the api server", func() {
			// given
			store := setupErrorStore(400, "some error from the server")

			// when
			resource := mesh.MeshResource{
				Spec: v1alpha1.Mesh{},
				Meta: &model.ResourceMeta{
					Mesh: "default",
					Name: "default",
				},
			}
			err := store.Create(context.Background(), &resource)

			// then
			Expect(err).To(MatchError("(400): some error from the server"))
		})

		It("should parse kuma api server error", func() {
			json := `
			{
				"title": "Could not process resource",
				"details": "Resource is not valid",
				"causes": [
					{
						"field": "mtls",
						"message": "cannot be empty"
					},
					{
						"field": "mesh",
						"message": "cannot be empty"
					}
				]
			}
		`
			store := setupErrorStore(400, json)

			// when
			resource := mesh.MeshResource{
				Meta: &model.ResourceMeta{
					Mesh: "test",
					Name: "test",
				},
			}
			err := store.Update(context.Background(), &resource)

			// then
			Expect(err).To(HaveOccurred())

			Expect(err).To(Equal(&errors_types.Error{
				Title:   "Could not process resource",
				Details: "Resource is not valid",
				Causes: []errors_types.Cause{
					{
						Field:   "mtls",
						Message: "cannot be empty",
					},
					{
						Field:   "mesh",
						Message: "cannot be empty",
					},
				},
			}))
		})
	})

	Describe("List()", func() {
		It("should successfully list known resources", func() {
			// given
			store := setupStore("list.json", func(req *http.Request) {
				Expect(req.URL.Path).To(Equal(fmt.Sprintf("/meshes/demo/traffic-routes")))
			})

			// when
			rs := sample_core.TrafficRouteResourceList{}
			err := store.List(context.Background(), &rs, core_store.ListByMesh("demo"))

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(rs.Items).To(HaveLen(2))
			// and
			Expect(rs.Items[0].Meta.GetName()).To(Equal("one"))
			Expect(rs.Items[0].Meta.GetMesh()).To(Equal("default"))
			Expect(rs.Items[0].Meta.GetVersion()).To(Equal(""))
			Expect(rs.Items[0].Spec.Path).To(Equal("/example"))
			// and
			Expect(rs.Items[1].Meta.GetName()).To(Equal("two"))
			Expect(rs.Items[1].Meta.GetMesh()).To(Equal("demo"))
			Expect(rs.Items[1].Meta.GetVersion()).To(Equal(""))
			Expect(rs.Items[1].Spec.Path).To(Equal("/another"))
		})

		It("should list known resources using pagination", func() {
			// given
			store := setupStore("list-pagination.json", func(req *http.Request) {
				Expect(req.URL.Path).To(Equal(fmt.Sprintf("/meshes/demo/traffic-routes")))
				Expect(req.URL.Query().Get("size")).To(Equal("1"))
				Expect(req.URL.Query().Get("offset")).To(Equal("2"))
			})

			// when
			rs := sample_core.TrafficRouteResourceList{}
			err := store.List(context.Background(), &rs, core_store.ListByMesh("demo"), core_store.ListByPage(1, "2"))

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(rs.Items).To(HaveLen(1))
			// and
			Expect(rs.Items[0].Meta.GetName()).To(Equal("one"))
			Expect(rs.Items[0].Meta.GetMesh()).To(Equal("default"))
			Expect(rs.Items[0].Meta.GetVersion()).To(Equal(""))
			Expect(rs.Items[0].Spec.Path).To(Equal("/example"))
		})

		It("should list meshes", func() {
			// given
			store := setupStore("list-meshes.json", func(req *http.Request) {
				Expect(req.URL.Path).To(Equal(fmt.Sprintf("/meshes")))
			})

			// when
			meshes := mesh.MeshResourceList{}
			err := store.List(context.Background(), &meshes)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(meshes.Items).To(HaveLen(2))

			Expect(meshes.Items[0].Meta.GetName()).To(Equal("mesh-1"))
			Expect(meshes.Items[0].Meta.GetMesh()).To(Equal("mesh-1"))

			Expect(meshes.Items[1].Meta.GetName()).To(Equal("mesh-2"))
			Expect(meshes.Items[1].Meta.GetMesh()).To(Equal("mesh-2"))
		})

		It("should return error from the api server", func() {
			// given
			store := setupErrorStore(400, "some error from the server")

			// when
			meshes := mesh.MeshResourceList{}
			err := store.List(context.Background(), &meshes)

			// then
			Expect(err).To(MatchError("(400): some error from the server"))
		})

		It("should parse kuma api server error", func() {
			json := `
			{
				"title": "Could not list resource",
				"details": "Internal Server Error"
			}
		`
			store := setupErrorStore(400, json)

			// when
			meshes := mesh.MeshResourceList{}
			err := store.List(context.Background(), &meshes)

			// then
			Expect(err).To(HaveOccurred())

			Expect(err).To(Equal(&errors_types.Error{
				Title:   "Could not list resource",
				Details: "Internal Server Error",
			}))
		})
	})

	Describe("Delete()", func() {
		It("should delete the resource", func() {
			// given
			name := "tr-1"
			meshName := "mesh-1"
			store := setupStore("delete.json", func(req *http.Request) {
				Expect(req.URL.Path).To(Equal(fmt.Sprintf("/meshes/%s/traffic-routes/%s", meshName, name)))
			})

			// when
			resource := sample_core.TrafficRouteResource{}
			err := store.Delete(context.Background(), &resource, core_store.DeleteByKey(name, meshName))

			// then
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete mesh resource", func() {
			// given
			meshName := "mesh-1"
			store := setupStore("delete.json", func(req *http.Request) {
				Expect(req.URL.Path).To(Equal(fmt.Sprintf("/meshes/%s", meshName)))
			})

			// when
			resource := mesh.MeshResource{}
			err := store.Delete(context.Background(), &resource, core_store.DeleteByKey(meshName, meshName))

			// then
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return error from the api server", func() {
			// given
			store := setupErrorStore(400, "some error from the server")

			// when
			resource := sample_core.TrafficRouteResource{}
			err := store.Delete(context.Background(), &resource, core_store.DeleteByKey("tr-1", "mesh-1"))

			// then
			Expect(err).To(MatchError("(400): some error from the server"))
		})

		It("should map 404 error to ResourceNotFound", func() {
			// given
			json := `
			{
				"title": "Could not get a resource",
				"details": "Not found"
			}`
			store := setupErrorStore(404, json)

			// when
			resource := sample_core.TrafficRouteResource{}
			err := store.Delete(context.Background(), &resource, core_store.DeleteByKey("tr-1", "mesh-1"))

			// then
			Expect(core_store.IsResourceNotFound(err)).To(BeTrue())
		})

		It("should parse kuma api server error", func() {
			json := `
			{
				"title": "Could not delete resource",
				"details": "Internal Server Error"
			}`
			store := setupErrorStore(400, json)

			// when
			resource := sample_core.TrafficRouteResource{}
			err := store.Delete(context.Background(), &resource, core_store.DeleteByKey("tr-1", "mesh-1"))

			// then
			Expect(err).To(HaveOccurred())

			Expect(err).To(Equal(&errors_types.Error{
				Title:   "Could not delete resource",
				Details: "Internal Server Error",
			}))
		})
	})

})

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
