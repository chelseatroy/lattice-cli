package app_runner_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/receptor"
	"github.com/cloudfoundry-incubator/receptor/fake_receptor"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	"github.com/pivotal-cf-experimental/lattice-cli/app_runner"
)

var _ = Describe("AppRunner", func() {

	var (
		fakeReceptorClient *fake_receptor.FakeClient
		appRunner          app_runner.AppRunner
	)

	BeforeEach(func() {
		fakeReceptorClient = &fake_receptor.FakeClient{}
		appRunner = app_runner.New(fakeReceptorClient, "myDiegoInstall.com")

	})

	Describe("StartDockerApp", func() {
		It("Starts a Docker App", func() {
			fakeReceptorClient.DesiredLRPsReturns([]receptor.DesiredLRPResponse{}, nil)

			args := []string{"app", "arg1", "--app", "arg 2"}
			envs := map[string]string{"APPROOT": "/root/env/path"}
			err := appRunner.StartDockerApp("americano-app", "docker://runtest/runner", "/app-run-statement", args, envs, true, 128, 1024, 2000)
			Expect(err).To(BeNil())

			Expect(fakeReceptorClient.CreateDesiredLRPCallCount()).To(Equal(1))
			Expect(fakeReceptorClient.CreateDesiredLRPArgsForCall(0)).To(Equal(receptor.DesiredLRPCreateRequest{
				ProcessGuid:          "americano-app",
				Domain:               "diego-edge",
				RootFSPath:           "docker://runtest/runner",
				Instances:            1,
				Stack:                "lucid64",
				EnvironmentVariables: []receptor.EnvironmentVariable{receptor.EnvironmentVariable{Name: "APPROOT", Value: "/root/env/path"}, receptor.EnvironmentVariable{Name: "PORT", Value: "2000"}},
				Routes:               []string{"americano-app.myDiegoInstall.com"},
				MemoryMB:             128,
				DiskMB:               1024,
				Ports:                []uint32{2000},
				LogGuid:              "americano-app",
				LogSource:            "APP",
				Setup: &models.DownloadAction{
					From: "http://file_server.service.dc1.consul:8080/v1/static/docker-circus/docker-circus.tgz",
					To:   "/tmp",
				},
				Action: &models.RunAction{
					Path:       "/app-run-statement",
					Args:       []string{"app", "arg1", "--app", "arg 2"},
					Privileged: true,
				},
				Monitor: &models.RunAction{
					Path:      "/tmp/spy",
					Args:      []string{"-addr", ":2000"},
					LogSource: "HEALTH",
				},
			}))
		})

		It("returns errors if the app is already desired", func() {
			desiredLRPs := []receptor.DesiredLRPResponse{receptor.DesiredLRPResponse{ProcessGuid: "app-already-desired", Instances: 1}}
			fakeReceptorClient.DesiredLRPsReturns(desiredLRPs, nil)

			err := appRunner.StartDockerApp("app-already-desired", "docker://faily/boom", "/app-bork-statement", []string{}, map[string]string{}, false, 128, 1024, 8080)

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("App app-already-desired, is already running"))
			Expect(fakeReceptorClient.DesiredLRPsCallCount()).To(Equal(1))
		})

		Describe("returning errors from the receptor", func() {
			It("returns desiring lrp errors", func() {
				receptorError := errors.New("error - Desiring an LRP")
				fakeReceptorClient.CreateDesiredLRPReturns(receptorError)

				err := appRunner.StartDockerApp("nescafe-app", "docker://faily/boom", "/app-bork-statement", []string{}, map[string]string{}, false, 128, 1024, 8080)
				Expect(err).To(Equal(receptorError))
			})

			It("returns existing count errors", func() {
				receptorError := errors.New("error - Existing Count")
				fakeReceptorClient.DesiredLRPsReturns([]receptor.DesiredLRPResponse{}, receptorError)

				err := appRunner.StartDockerApp("nescafe-app", "docker://faily/boom", "/app-bork-statement", []string{}, map[string]string{}, false, 128, 1024, 8080)
				Expect(err).To(Equal(receptorError))
			})
		})

	})

	Describe("ScaleApp", func() {

		It("Scales a Docker App", func() {
			desiredLRPs := []receptor.DesiredLRPResponse{receptor.DesiredLRPResponse{ProcessGuid: "americano-app", Instances: 1}}
			fakeReceptorClient.DesiredLRPsReturns(desiredLRPs, nil)
			instanceCount := 25

			err := appRunner.ScaleApp("americano-app", instanceCount)
			Expect(err).To(BeNil())

			Expect(fakeReceptorClient.UpdateDesiredLRPCallCount()).To(Equal(1))
			processGuid, updateRequest := fakeReceptorClient.UpdateDesiredLRPArgsForCall(0)
			Expect(processGuid).To(Equal("americano-app"))

			Expect(updateRequest).To(Equal(receptor.DesiredLRPUpdateRequest{Instances: &instanceCount}))
		})

		It("returns errors if the app is NOT already started", func() {
			desiredLRPs := []receptor.DesiredLRPResponse{receptor.DesiredLRPResponse{ProcessGuid: "americano-app", Instances: 1}}
			fakeReceptorClient.DesiredLRPsReturns(desiredLRPs, nil)

			err := appRunner.ScaleApp("app-not-running", 15)

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("app-not-running, is not started. Please start an app first"))
			Expect(fakeReceptorClient.DesiredLRPsCallCount()).To(Equal(1))
		})

		Describe("returning errors from the receptor", func() {
			It("returns desiring lrp errors", func() {
				desiredLRPs := []receptor.DesiredLRPResponse{receptor.DesiredLRPResponse{ProcessGuid: "americano-app", Instances: 1}}
				fakeReceptorClient.DesiredLRPsReturns(desiredLRPs, nil)

				receptorError := errors.New("error - Updating an LRP")
				fakeReceptorClient.UpdateDesiredLRPReturns(receptorError)

				err := appRunner.ScaleApp("americano-app", 17)
				Expect(err).To(Equal(receptorError))
			})

			It("returns errors fetching the existing lrp count", func() {
				receptorError := errors.New("error - Existing Count")
				fakeReceptorClient.DesiredLRPsReturns([]receptor.DesiredLRPResponse{}, receptorError)

				err := appRunner.ScaleApp("nescafe-app", 2)
				Expect(err).To(Equal(receptorError))
			})
		})

	})

	Describe("RemoveApp", func() {
		It("Removes a Docker App", func() {
			desiredLRPs := []receptor.DesiredLRPResponse{receptor.DesiredLRPResponse{ProcessGuid: "americano-app", Instances: 1}}
			fakeReceptorClient.DesiredLRPsReturns(desiredLRPs, nil)
			fakeReceptorClient.DeleteDesiredLRPReturns(nil)

			err := appRunner.RemoveApp("americano-app")
			Expect(err).To(BeNil())

			Expect(fakeReceptorClient.DeleteDesiredLRPCallCount()).To(Equal(1))
			Expect(fakeReceptorClient.DeleteDesiredLRPArgsForCall(0)).To(Equal("americano-app"))
		})

		It("returns errors if the app is NOT already started", func() {
			desiredLRPs := []receptor.DesiredLRPResponse{receptor.DesiredLRPResponse{ProcessGuid: "americano-app", Instances: 1}}
			fakeReceptorClient.DesiredLRPsReturns(desiredLRPs, nil)

			err := appRunner.RemoveApp("app-not-running")

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("app-not-running, is not started. Please start an app first"))
			Expect(fakeReceptorClient.DesiredLRPsCallCount()).To(Equal(1))

		})

		Describe("returning errors from the receptor", func() {
			It("returns deleting lrp errors", func() {
				desiredLRPs := []receptor.DesiredLRPResponse{receptor.DesiredLRPResponse{ProcessGuid: "americano-app", Instances: 1}}
				fakeReceptorClient.DesiredLRPsReturns(desiredLRPs, nil)

				deletingError := errors.New("deleting failed")
				fakeReceptorClient.DeleteDesiredLRPReturns(deletingError)

				err := appRunner.RemoveApp("americano-app")

				Expect(err).To(Equal(deletingError))
			})

			It("returns errors fetching the existing count", func() {
				receptorError := errors.New("error - Existing Count")
				fakeReceptorClient.DesiredLRPsReturns([]receptor.DesiredLRPResponse{}, receptorError)

				err := appRunner.RemoveApp("nescafe-app")
				Expect(err).To(Equal(receptorError))
			})
		})

	})

	Describe("IsAppUp", func() {
		It("returns true if the docker app is running with a status of ActualLRPStateRunning", func() {
			actualLrpsResponse := []receptor.ActualLRPResponse{receptor.ActualLRPResponse{ProcessGuid: "americano-app", State: receptor.ActualLRPStateRunning}}
			fakeReceptorClient.ActualLRPsByProcessGuidReturns(actualLrpsResponse, nil)

			status, err := appRunner.IsAppUp("americano-app")
			Expect(err).To(BeNil())
			Expect(status).To(BeTrue())

			Expect(fakeReceptorClient.ActualLRPsByProcessGuidCallCount()).To(Equal(1))

			Expect(fakeReceptorClient.ActualLRPsByProcessGuidArgsForCall(0)).To(Equal("americano-app"))
		})

		It("returns false if the docker app is running without a status of ActualLRPStateRunning", func() {
			actualLrpsResponse := []receptor.ActualLRPResponse{receptor.ActualLRPResponse{ProcessGuid: "americano-app", State: receptor.ActualLRPStateClaimed}}
			fakeReceptorClient.ActualLRPsByProcessGuidReturns(actualLrpsResponse, nil)

			status, _ := appRunner.IsAppUp("americano-app")
			Expect(status).To(BeFalse())

		})

		It("returns false if the docker app is not running", func() {
			actualLrpsResponse := []receptor.ActualLRPResponse{}
			fakeReceptorClient.ActualLRPsByProcessGuidReturns(actualLrpsResponse, nil)

			status, _ := appRunner.IsAppUp("americano-app")
			Expect(status).To(BeFalse())

		})

		Describe("returning errors from the receptor", func() {
			It("returns errors fetching the existing count", func() {
				receptorError := errors.New("error - Existing Count")
				fakeReceptorClient.ActualLRPsByProcessGuidReturns([]receptor.ActualLRPResponse{}, receptorError)
				_, err := appRunner.IsAppUp("nescafe-app")
				Expect(err).To(Equal(receptorError))
			})
		})
	})

	Describe("AppExists", func() {
		It("returns true if the docker app exists", func() {
			desiredLRPs := []receptor.DesiredLRPResponse{receptor.DesiredLRPResponse{ProcessGuid: "americano-app", Instances: 1}}
			fakeReceptorClient.DesiredLRPsReturns(desiredLRPs, nil)

			exists, err := appRunner.AppExists("americano-app")
			Expect(err).To(BeNil())
			Expect(exists).To(Equal(true))
		})

		It("returns false if the docker app does not exist", func() {
			desiredLRPs := []receptor.DesiredLRPResponse{}
			fakeReceptorClient.DesiredLRPsReturns(desiredLRPs, nil)

			exists, err := appRunner.AppExists("americano-app")
			Expect(err).To(BeNil())
			Expect(exists).To(Equal(false))
		})

		Describe("returning errors from the receptor", func() {
			It("returns errors fetching the status", func() {
				desiredLRPs := []receptor.DesiredLRPResponse{}
				fakeReceptorClient.DesiredLRPsReturns(desiredLRPs, errors.New("Something Bad"))

				exists, err := appRunner.AppExists("americano-app")
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal("Something Bad"))
				Expect(exists).To(Equal(false))
			})
		})
	})
})
