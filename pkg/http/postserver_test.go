package http

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/inject"
	mock_client "github.com/bakito/batch-job-controller/pkg/mocks/client"
	mock_lifecycle "github.com/bakito/batch-job-controller/pkg/mocks/lifecycle"
	mock_logr "github.com/bakito/batch-job-controller/pkg/mocks/logr"
	mock_record "github.com/bakito/batch-job-controller/pkg/mocks/record"
	"github.com/bakito/batch-job-controller/pkg/test"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gm "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	reportJSON              = `{ "test": [{ "value": 1.0, "labels": { "label_a": "AAA", "label_b": "BBB" }}] }`
	eventMessageJSON        = `{ "warning": true, "reason": "TestReason", "message": "test message" }`
	eventMessageInvalidJSON = `{ "warning": true, "reason": "testReason", "message": "test message" }`
	eventMessageArgsJSON    = `{ "warning": true, "reason": "TestReason", "message": "test message: %s" ,"args" : ["a1"]}`
)

var (
	_ inject.Healthz       = &PostServer{}
	_ inject.EventRecorder = &PostServer{}
	_ inject.Config        = &PostServer{}
	_ inject.Controller    = &PostServer{}
	_ inject.Reader        = &PostServer{}
)

var _ = Describe("HTTP", func() {
	var (
		mockCtrl       *gm.Controller // gomock struct
		mockSink       *mock_logr.MockLogSink
		mockController *mock_lifecycle.MockController
		mockReader     *mock_client.MockReader
		executionID    string
		node           string

		s   *PostServer
		cfg *config.Config

		rr     *httptest.ResponseRecorder
		router *gin.Engine
		path   string
	)
	BeforeEach(func() {
		gin.SetMode(gin.ReleaseMode)
		mockCtrl = gm.NewController(GinkgoT())
		mockSink = mock_logr.NewMockLogSink(mockCtrl)
		mockReader = mock_client.NewMockReader(mockCtrl)
		mockController = mock_lifecycle.NewMockController(mockCtrl)
		executionID = uuid.New().String()
		node = uuid.New().String()
		tmp, err := test.TempDir(executionID)
		Ω(err).ShouldNot(HaveOccurred())
		cfg = &config.Config{
			Metrics: config.Metrics{
				Prefix: "foo",
			},
			ReportDirectory: tmp,
		}

		mockSink.EXPECT().Init(gm.Any())
		mockSink.EXPECT().Enabled(gm.Any()).AnyTimes().Return(true)
		s = &PostServer{
			Config: cfg,
			Server: &Server{
				Log:    logr.New(mockSink),
				Config: cfg,
			},
		}
		s.InjectReader(mockReader)
		s.InjectController(mockController)
		s.InjectConfig(cfg)

		rr = httptest.NewRecorder()

		// Need to create a router that we can pass the request through so that the vars will be added to the context
		router = gin.New()
		path = fmt.Sprintf("/report/%s/%s%s", node, executionID, CallbackBaseResultSubPath)
		DeferCleanup(func() error {
			return os.RemoveAll(s.Config.ReportDirectory)
		})
	})
	Context("postResult", func() {
		BeforeEach(func() {
			router.POST(CallbackBasePath+CallbackBaseResultSubPath, s.postResult)

			mockSink.EXPECT().WithValues("node", node, "id", executionID).Return(mockSink)
			mockSink.EXPECT().WithValues("length", gm.Any()).Return(mockSink)
		})
		It("succeed if file is saved", func() {
			mockController.EXPECT().ReportReceived(executionID, node, gm.Any(), gm.Any())
			mockSink.EXPECT().WithValues("name", gm.Any(), "path", gm.Any()).Return(mockSink)
			mockSink.EXPECT().Info(gm.Any(), "received results")

			req, err := http.NewRequest("POST", path, strings.NewReader(reportJSON))
			Ω(err).ShouldNot(HaveOccurred())

			router.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusOK))

			files, err := os.ReadDir(filepath.Join(s.Config.ReportDirectory, executionID))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(files).Should(HaveLen(1))

			b, err := os.ReadFile(filepath.Join(s.Config.ReportDirectory, executionID, files[0].Name()))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(b).Should(Equal([]byte(reportJSON)))
		})
		It("fails if json is invalid", func() {
			mockSink.EXPECT().Error(gm.Any(), gm.Any())

			req, err := http.NewRequest("POST", path, strings.NewReader("foo"))
			Ω(err).ShouldNot(HaveOccurred())

			router.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusBadRequest))

			files, err := os.ReadDir(filepath.Join(s.Config.ReportDirectory, executionID))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(files).Should(HaveLen(0))
		})
	})

	Context("middleware", func() {
		var handler *testing.FakeHandler
		BeforeEach(func() {
			handler = &testing.FakeHandler{
				StatusCode: 200,
			}
			router.Use(s.middleware)
			router.POST(CallbackBasePath+CallbackBaseResultSubPath, gin.WrapH(handler))
		})

		It("should allow the request", func() {
			mockController.EXPECT().Has(node, executionID).Return(true)

			req, err := http.NewRequest("POST", path, strings.NewReader(""))
			Ω(err).ShouldNot(HaveOccurred())

			router.ServeHTTP(rr, req)

			handler.ValidateRequestCount(GinkgoT(), 1)
		})
		It("should allow the request if controller is nil", func() {
			s.InjectController(nil)
			req, err := http.NewRequest("POST", path, strings.NewReader(""))
			Ω(err).ShouldNot(HaveOccurred())

			router.ServeHTTP(rr, req)

			handler.ValidateRequestCount(GinkgoT(), 1)
		})
		It("should deny if execution is not known", func() {
			mockController.EXPECT().Has(node, executionID).Return(false)

			req, err := http.NewRequest("POST", path, strings.NewReader(""))
			Ω(err).ShouldNot(HaveOccurred())

			router.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusNotAcceptable))
			Ω(rr.Body.String()).Should(HavePrefix(errorMiddlewareNotAcceptable))
			handler.ValidateRequestCount(GinkgoT(), 0)
		})
	})

	Context("postFile", func() {
		var path string
		BeforeEach(func() {
			path = fmt.Sprintf("/report/%s/%s%s", node, executionID, CallbackBaseFileSubPath)
			router.POST(CallbackBasePath+CallbackBaseFileSubPath, s.postFile)
			mockSink.EXPECT().WithValues("node", node, "id", executionID).Return(mockSink)
		})
		Context("single file", func() {
			var (
				fileName               string
				generatedFileExtension string
			)
			BeforeEach(func() {
				fileName = uuid.New().String() + ".txt"

				mockSink.EXPECT().WithValues("name", gm.Any(), "path", gm.Any(), "length", gm.Any()).Return(mockSink)
				mockSink.EXPECT().Info(gm.Any(), "received 1 file")
				DeferCleanup(func() error {
					Ω(rr.Code).Should(Equal(http.StatusOK))

					files, err := os.ReadDir(filepath.Join(s.Config.ReportDirectory, executionID))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(files).Should(HaveLen(1))
					if generatedFileExtension != "" {
						Ω(files[0].Name()).Should(HavePrefix(node + "-"))
						Ω(files[0].Name()).Should(HaveSuffix(generatedFileExtension))
					} else {
						Ω(files[0].Name()).Should(Equal(node + "-" + fileName))
					}

					b, err := os.ReadFile(filepath.Join(s.Config.ReportDirectory, executionID, files[0].Name()))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(b).Should(Equal([]byte("foo")))
					return nil
				})
			})
			It("succeed if file is saved with correct name from query parameter", func() {
				req, err := http.NewRequest("POST", fmt.Sprintf("%s?name=%s", path, fileName), strings.NewReader("foo"))
				Ω(err).ShouldNot(HaveOccurred())
				router.ServeHTTP(rr, req)
			})
			It("succeed if file is saved with correct name from header", func() {
				req, err := http.NewRequest("POST", path, strings.NewReader("foo"))
				req.Header.Add("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, fileName))
				Ω(err).ShouldNot(HaveOccurred())
				router.ServeHTTP(rr, req)
			})
			It("succeed if file is saved with generated name with .file extension", func() {
				generatedFileExtension = ".file"
				req, err := http.NewRequest("POST", path, strings.NewReader("foo"))
				Ω(err).ShouldNot(HaveOccurred())
				router.ServeHTTP(rr, req)
			})
			It("succeed if file is saved with generated name with .txt extension", func() {
				generatedFileExtension = ".txt"
				req, err := http.NewRequest("POST", path, strings.NewReader("foo"))
				req.Header.Add("Content-Type", "text/plain")
				Ω(err).ShouldNot(HaveOccurred())
				router.ServeHTTP(rr, req)
			})
			It("succeed if file is saved with generated name with .json extension", func() {
				generatedFileExtension = ".json"
				req, err := http.NewRequest("POST", path, strings.NewReader("foo"))
				req.Header.Add("content-type", "application/json")
				Ω(err).ShouldNot(HaveOccurred())
				router.ServeHTTP(rr, req)
			})
		})
		Context("multiple files", func() {
			It("upload 2 files", func() {
				mockSink.EXPECT().WithValues("names", gm.Any()).Return(mockSink)
				mockSink.EXPECT().Info(gm.Any(), "received 2 file(s)")

				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part1, _ := writer.CreateFormFile("file", filepath.Base("a"))
				_, _ = io.Copy(part1, strings.NewReader("file a"))
				part2, _ := writer.CreateFormFile("file", filepath.Base("b"))
				_, _ = io.Copy(part2, strings.NewReader("file b"))
				_ = writer.Close()

				req, _ := http.NewRequest("POST", path, body)
				req.Header.Add("Content-Type", writer.FormDataContentType())

				router.ServeHTTP(rr, req)
			})
		})
	})
	Context("postEvent", func() {
		var (
			path       string
			mockRecord *mock_record.MockEventRecorder
		)
		BeforeEach(func() {
			mockRecord = mock_record.NewMockEventRecorder(mockCtrl)
			s.InjectEventRecorder(mockRecord)
			path = fmt.Sprintf("/report/%s/%s%s", node, executionID, CallbackBaseEventSubPath)
			router.POST(CallbackBasePath+CallbackBaseEventSubPath, s.postEvent)
		})
		It("succeed if event with message is sent", func() {
			mockSink.EXPECT().WithValues("node", node, "id", executionID).Return(mockSink)
			mockSink.EXPECT().WithValues("length", gm.Any()).Return(mockSink)
			mockSink.EXPECT().WithValues("pod", gm.Any(), "type", "Warning", "reason", "TestReason", "event-message", "test message").Return(mockSink)
			mockRecord.EXPECT().Event(gm.Any(), "Warning", "TestReason", "test message")
			mockSink.EXPECT().Info(gm.Any(), "event created")
			mockReader.EXPECT().
				Get(gm.Any(), client.ObjectKey{Namespace: s.Config.Namespace, Name: s.Config.PodName(node, executionID)}, gm.AssignableToTypeOf(&corev1.Pod{}))

			req, err := http.NewRequest("POST", path, strings.NewReader(eventMessageJSON))
			Ω(err).ShouldNot(HaveOccurred())

			router.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusOK))
		})
		It("succeed if event with message with args is sent", func() {
			mockSink.EXPECT().WithValues("node", node, "id", executionID).Return(mockSink)
			mockSink.EXPECT().WithValues("length", gm.Any()).Return(mockSink)
			mockSink.EXPECT().WithValues("pod", gm.Any(), "type", "Warning", "reason", "TestReason", "event-message", "test message: a1").Return(mockSink)
			mockRecord.EXPECT().Eventf(gm.Any(), "Warning", "TestReason", "test message: %s", "a1")
			mockSink.EXPECT().Info(gm.Any(), "event created")
			mockReader.EXPECT().
				Get(gm.Any(), client.ObjectKey{Namespace: s.Config.Namespace, Name: s.Config.PodName(node, executionID)}, gm.AssignableToTypeOf(&corev1.Pod{}))

			req, err := http.NewRequest("POST", path, strings.NewReader(eventMessageArgsJSON))
			Ω(err).ShouldNot(HaveOccurred())

			router.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusOK))
		})

		It("fails if json is invalid", func() {
			mockSink.EXPECT().WithValues("node", node, "id", executionID).Return(mockSink)
			mockSink.EXPECT().WithValues("length", gm.Any()).Return(mockSink)
			mockSink.EXPECT().WithValues("event", gm.Any()).Return(mockSink)
			mockSink.EXPECT().Error(gm.Any(), gm.Any())

			req, err := http.NewRequest("POST", path, strings.NewReader("foo"))
			Ω(err).ShouldNot(HaveOccurred())

			router.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusBadRequest))
			Ω(rr.Body.String()).Should(HavePrefix("error decoding event"))
		})

		It("fails if event is invalid", func() {
			mockSink.EXPECT().WithValues("node", node, "id", executionID).Return(mockSink)
			mockSink.EXPECT().WithValues("length", gm.Any()).Return(mockSink)
			mockSink.EXPECT().Error(gm.Any(), "event is invalid")

			req, err := http.NewRequest("POST", path, strings.NewReader(eventMessageInvalidJSON))
			Ω(err).ShouldNot(HaveOccurred())

			router.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusBadRequest))
			Ω(rr.Body.String()).Should(ContainSubstring("'Reason' failed on the 'first_char_must_be_uppercase' tag"))
		})

		It("fails if pod not found", func() {
			mockSink.EXPECT().WithValues("node", node, "id", executionID).Return(mockSink)
			mockSink.EXPECT().WithValues("length", gm.Any()).Return(mockSink)
			mockSink.EXPECT().Error(gm.Any(), gm.Any())
			mockReader.EXPECT().
				Get(gm.Any(), client.ObjectKey{Namespace: s.Config.Namespace, Name: s.Config.PodName(node, executionID)}, gm.AssignableToTypeOf(&corev1.Pod{})).
				Return(fmt.Errorf("error"))

			req, err := http.NewRequest("POST", path, strings.NewReader(eventMessageArgsJSON))
			Ω(err).ShouldNot(HaveOccurred())

			router.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusNotFound))
			Ω(strings.TrimSpace(rr.Body.String())).Should(HavePrefix("error finding pod"))
		})
	})

	Context("StaticFileServer", func() {
		It("returns a file server", func() {
			cfg.ReportDirectory = "path"
			sfs := StaticFileServer(1234, cfg)
			Ω(sfs).ShouldNot(BeNil())
			Ω(sfs.(*Server).Port).Should(Equal(1234))
			Ω(sfs.(*Server).Kind).Should(Equal("public"))
			Ω(sfs.(*Server).Handler).ShouldNot(BeNil())
		})
	})

	Context("GenericAPIServer", func() {
		It("returns a server", func() {
			cfg.ReportDirectory = ""
			sfs := GenericAPIServer(1234, cfg)
			Ω(sfs).ShouldNot(BeNil())
			Ω(sfs.(*PostServer).Port).Should(Equal(1234))
			Ω(sfs.(*PostServer).Kind).Should(Equal("internal"))
		})
	})

	Context("MockAPIServer", func() {
		It("returns a server", func() {
			sfs := MockAPIServer(1234)
			Ω(sfs).ShouldNot(BeNil())
			Ω(sfs.(*mockServer).Port).Should(Equal(1234))
			Ω(sfs.(*mockServer).Kind).Should(Equal("internal"))
		})
	})
})
