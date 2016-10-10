package cephbroker_test

import (
	"encoding/json"

	"code.cloudfoundry.org/cephbroker/cephbroker"
	"code.cloudfoundry.org/cephbroker/cephfakes"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/voldriver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"context"
	"code.cloudfoundry.org/voldriver/driverhttp"
)

var Controller = Describe("Controller", func() {
	var (
		logger     lager.Logger
		ctx					context.Context
		env					voldriver.Env
		fakeClient cephbroker.Client
		subject    cephbroker.Controller
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test-broker")
		ctx = context.TODO()
		env = driverhttp.NewHttpDriverEnv(&logger, &ctx)
		fakeClient = &cephfakes.FakeClient{}
		subject = cephbroker.NewController(fakeClient)
	})
	Context(".Create", func() {
		It("should be able to create mount", func() {
			resp := subject.Create(env, voldriver.CreateRequest{
				Name: "InstanceID",
			})
			Expect(resp.Err).To(Equal(""))
		})
	})
	Context(".Remove", func() {
		It("should be able to remove mount", func() {
			resp := subject.Remove(env, voldriver.RemoveRequest{Name: "InstanceId"})
			Expect(resp.Err).To(Equal(""))
		})
	})
	Context(".Bind", func() {
		It("should be able to bind", func() {
			resp := subject.Bind(env, "InstanceId")
			Expect(resp.Err).To(Equal(""))
			Expect(json.Marshal(resp)).To(ContainSubstring(
				"{\"Err\":\"\",\"SharedDevice\":{\"volume_id\":\"InstanceId\",\"mount_config\":" +
					"{\"ip\":\"\",\"keyring\":\"\",\"local_mount_point\":\"\",\"remote_mount_point\":\"\"}}}",
			))
		})
	})
})
