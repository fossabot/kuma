package kumainjector_test

import (
	"io/ioutil"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Kong/kuma/pkg/config"
	kuma_injector "github.com/Kong/kuma/pkg/config/app/kuma-injector"
)

var _ = Describe("Config", func() {

	It("should be loadable from configuration file", func() {
		// given
		cfg := kuma_injector.Config{}

		// when
		err := config.Load(filepath.Join("testdata", "valid-config.input.yaml"), &cfg)

		// then
		Expect(err).ToNot(HaveOccurred())

		// and
		Expect(cfg.WebHookServer.Address).To(Equal("127.0.0.2"))
		Expect(cfg.WebHookServer.Port).To(Equal(uint32(8442)))
		Expect(cfg.WebHookServer.CertDir).To(Equal("/var/secret/kuma-injector"))
		// and
		Expect(cfg.Injector.ControlPlane.ApiServer.URL).To(Equal("https://api-server:8765"))
		// and
		Expect(cfg.Injector.SidecarContainer.Image).To(Equal("kuma-sidecar:latest"))
		Expect(cfg.Injector.SidecarContainer.RedirectPort).To(Equal(uint32(1234)))
		Expect(cfg.Injector.SidecarContainer.UID).To(Equal(int64(2345)))
		Expect(cfg.Injector.SidecarContainer.GID).To(Equal(int64(3456)))
		Expect(cfg.Injector.SidecarContainer.AdminPort).To(Equal(uint32(45678)))
		Expect(cfg.Injector.SidecarContainer.DrainTime).To(Equal(15 * time.Second))
		// and
		Expect(cfg.Injector.SidecarContainer.ReadinessProbe.InitialDelaySeconds).To(Equal(int32(11)))
		Expect(cfg.Injector.SidecarContainer.ReadinessProbe.TimeoutSeconds).To(Equal(int32((13))))
		Expect(cfg.Injector.SidecarContainer.ReadinessProbe.PeriodSeconds).To(Equal(int32((15))))
		Expect(cfg.Injector.SidecarContainer.ReadinessProbe.SuccessThreshold).To(Equal(int32((11))))
		Expect(cfg.Injector.SidecarContainer.ReadinessProbe.FailureThreshold).To(Equal(int32((112))))
		// and
		Expect(cfg.Injector.SidecarContainer.LivenessProbe.InitialDelaySeconds).To(Equal(int32(260)))
		Expect(cfg.Injector.SidecarContainer.LivenessProbe.TimeoutSeconds).To(Equal(int32(23)))
		Expect(cfg.Injector.SidecarContainer.LivenessProbe.PeriodSeconds).To(Equal(int32(25)))
		Expect(cfg.Injector.SidecarContainer.LivenessProbe.FailureThreshold).To(Equal(int32(212)))
		// and
		Expect(cfg.Injector.SidecarContainer.Resources.Requests.CPU).To(Equal("150m"))
		Expect(cfg.Injector.SidecarContainer.Resources.Requests.Memory).To(Equal("164Mi"))
		Expect(cfg.Injector.SidecarContainer.Resources.Limits.CPU).To(Equal("1100m"))
		Expect(cfg.Injector.SidecarContainer.Resources.Limits.Memory).To(Equal("1512Mi"))
		// and
		Expect(cfg.Injector.InitContainer.Image).To(Equal("kuma-init:latest"))
		Expect(cfg.Injector.InitContainer.Enabled).To(Equal(false))
	})

	It("should have consistent defaults", func() {
		// given
		cfg := kuma_injector.DefaultConfig()

		// when
		actual, err := config.ToYAML(&cfg)
		// then
		Expect(err).ToNot(HaveOccurred())

		// when
		expected, err := ioutil.ReadFile(filepath.Join("testdata", "default-config.golden.yaml"))
		// then
		Expect(err).ToNot(HaveOccurred())
		// and
		Expect(actual).To(MatchYAML(expected))
	})

	It("should have validators", func() {
		// given
		cfg := kuma_injector.Config{}

		// when
		err := config.Load(filepath.Join("testdata", "invalid-config.input.yaml"), &cfg)

		// then
		Expect(err).To(MatchError(`Invalid configuration: .WebHookServer is not valid: .Address must be either empty or a valid IPv4/IPv6 address; .Port must be in the range [0, 65535]; .CertDir must be non-empty; .Injector is not valid: .ControlPlane is not valid: .ApiServer is not valid: .URL must be a valid absolute URI; .SidecarContainer is not valid: .Image must be non-empty; .RedirectPort must be in the range [0, 65535]; .AdminPort must be in the range [0, 65535]; .DrainTime must be positive; .ReadinessProbe is not valid: .InitialDelaySeconds must be >= 1; .TimeoutSeconds must be >= 1; .PeriodSeconds must be >= 1; .SuccessThreshold must be >= 1; .FailureThreshold must be >= 1; .LivenessProbe is not valid: .InitialDelaySeconds must be >= 1; .TimeoutSeconds must be >= 1; .PeriodSeconds must be >= 1; .FailureThreshold must be >= 1; .Resources is not valid: .Requests is not valid: .CPU is not valid: quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'; .Memory is not valid: quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'; .Limits is not valid: .CPU is not valid: quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'; .Memory is not valid: quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'; .InitContainer is not valid: .Image must be non-empty`))
	})
})
